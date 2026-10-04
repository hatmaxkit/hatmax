// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"net/url"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/mailer"
)

// MailboxDelivery owns trusted links and bounded application mail dispatch.
// It starts no worker and never propagates provider errors containing mail bodies.
type MailboxDelivery struct {
	service *core.RecoveryService
	queries *Queries
	sender  mailer.Mailer
	origin  string
	logger  log.Logger
}

func NewMailboxDelivery(service *core.RecoveryService, queries *Queries, sender mailer.Mailer, logger log.Logger, origin string, localhost bool) (*MailboxDelivery, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") || len(origin) > 2048 {
		return nil, errors.New("invalid trusted recovery origin")
	}

	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && localhost && local) {
		return nil, errors.New("recovery origin requires HTTPS")
	}

	if service == nil || queries == nil || sender == nil || logger == nil {
		return nil, errors.New("mailbox delivery dependencies are required")
	}

	if _, noop := sender.(*mailer.NoopMailer); noop {
		return nil, errors.New("mailbox delivery requires an active mailer")
	}

	u.Path = ""

	return &MailboxDelivery{service: service, queries: queries, sender: sender, logger: logger, origin: u.String()}, nil
}

// RequestMailboxVerification sends only the token committed for the exact mailbox.
// The HTTP owner groups every result into a fixed-time neutral acknowledgment.
func (d *MailboxDelivery) RequestMailboxVerification(ctx context.Context, mailbox string) error {
	address, err := mail.ParseAddress(mailbox)
	if err != nil || address.Address != mailbox || address.Name != "" {
		return core.ErrRecoveryUnavailable
	}

	work, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	issue, err := d.service.RequestMailboxVerification(work, mailbox)
	if err != nil {
		return err
	}

	link := d.origin + "/account/mailbox/confirm#token=" + issue.Token.Bearer()

	err = d.sender.Send(work, &mailer.Message{To: []mailer.Address{{Email: issue.Target}}, Subject: "Verify your Ticked mailbox", Text: "Open this link and submit the verification form:\n" + link + "\nThis link does not sign you in."})
	if err != nil {
		return errors.New("mailbox dispatch failed")
	}

	return nil
}

func (d *MailboxDelivery) ConfirmMailboxVerification(ctx context.Context, bearer string) error {
	work, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := d.service.ConfirmMailboxVerification(work, bearer)
	if err != nil {
		return err
	}
	// Delivery failure cannot undo security state or require another token consume.
	dispatchErr := d.DispatchMailboxNotices(work, result.Subject, 5)
	if dispatchErr != nil {
		d.logger.Error("Mailbox notification dispatch failed; committed security state is unchanged")
	}

	return nil
}

// DispatchMailboxNotices is an explicit application retry entrypoint: at most
// twenty intents, five attempts each, seven-day retention and one five-second call.
// Concurrent calls may duplicate delivery; durable attempts bound that behavior.
func (d *MailboxDelivery) DispatchMailboxNotices(ctx context.Context, subject string, limit int) error {
	if limit < 1 || limit > 20 {
		return core.ErrRecoveryCapacity
	}

	work, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	notices, err := d.queries.claimMailboxNotices(work, subject, limit)
	if err != nil {
		return err
	}

	var failed bool

	for _, n := range notices {
		err = d.sender.Send(work, &mailer.Message{To: []mailer.Address{{Email: n.Destination}}, Subject: "Your Ticked mailbox was verified", Text: "Your current mailbox was verified. Existing sessions were revoked. Sign in again through the normal authentication flow."})
		if err != nil {
			failed = true

			continue
		}

		err = dal.New(d.queries.dbProvider.GetDB()).DeliverRecoveryNotice(work, n.ID)
		if err != nil {
			failed = true
		}
	}

	if failed {
		return errors.New("mailbox notification dispatch failed")
	}

	return nil
}

func (q *Queries) claimMailboxNotices(ctx context.Context, subject string, limit int) ([]dal.RecoveryNotice, error) {
	if limit < 1 || limit > 20 {
		return nil, core.ErrRecoveryCapacity
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	_, err = queries.GetUserForAuth(ctx, subject)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, core.ErrRecoveryUnavailable
	}

	if err != nil {
		return nil, err
	}

	_, err = queries.ReclaimRecoveryNotices(ctx, subject)
	if err != nil {
		return nil, err
	}

	notices, err := queries.LockRecoveryNotices(ctx, dal.LockRecoveryNoticesParams{UserID: subject, PageLimit: int32(limit)})
	if err != nil {
		return nil, err
	}

	for _, n := range notices {
		if n.Kind != 1 {
			return nil, core.ErrRecoveryUnavailable
		}

		err = queries.AttemptRecoveryNotice(ctx, n.ID)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return notices, nil
}
