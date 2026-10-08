// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

  const {targetId} = await call('Target.createTarget', {url: 'about:blank'});
  const {sessionId} = await call('Target.attachToTarget', {targetId, flatten: true});
  const page = (method, params) => call(method, params, sessionId);
  await page('Page.enable');
  await page('Runtime.enable');
  await page('Network.enable');
  await page('WebAuthn.enable', {enableUI: false});
  const device = {
    protocol: 'ctap2', transport: 'internal', hasResidentKey: true,
    hasUserVerification: true, isUserVerified: true, automaticPresenceSimulation: true,
  };
  const {authenticatorId} = await page('WebAuthn.addVirtualAuthenticator', {options: device});
  const evaluate = async expression => {
    const result = await page('Runtime.evaluate', {expression, awaitPromise: true, returnByValue: true});
    // Never print exception text: it may contain transient protocol material.
    const exception = result.exceptionDetails?.exception;
    const firstLine = exception?.description?.split('\n')[0] || '';
    const safeDetail = /^Error: \/authenticators\/[a-z/-]+$/.test(firstLine) ? firstLine : exception?.className || 'unknown';
    assert(!result.exceptionDetails, `browser journey failed (${safeDetail}; inspect named step)`);
    return result.result.value;
  };
  const install = async () => evaluate(`(() => {
    const check = (condition, message) => {if (!condition) throw Error(message)};
    const decode = value => Uint8Array.from(atob(value.replace(/-/g,'+').replace(/_/g,'/')), c => c.charCodeAt(0));
    const encode = value => btoa(String.fromCharCode(...new Uint8Array(value))).replace(/\\+/g,'-').replace(/\\//g,'_').replace(/=+$/,'');
    const request = async (path, body, headers = {}) => {
      const r = await fetch(path, body === undefined ? {} : {method:'POST', headers:{'Content-Type':'application/json', ...headers}, body:JSON.stringify(body)});
      const text = await r.text();
      return {status:r.status, cache:r.headers.get('Cache-Control'), text, data:r.ok && text ? JSON.parse(text) : null};
    };
    const ok = async (path, body, headers) => {const r = await request(path, body, headers); check(r.status === 200, path); check(r.cache === 'no-store', 'cache'); return r.data};
    const denied = async (path, body, headers) => {const r = await request(path, body, headers); check(r.status === 403, path); return true};
    const password = async email => {const r = await fetch('/signin', {method:'POST', body:new URLSearchParams({email,password:'a distinct safe password'})}); check(r.status === 200 && r.headers.get('HX-Redirect') === '/list-items', 'password');};
    const create = async begin => {
      const publicKey = structuredClone(begin.options.publicKey);
      publicKey.challenge = decode(publicKey.challenge); publicKey.user.id = decode(publicKey.user.id);
      publicKey.excludeCredentials?.forEach(c => c.id = decode(c.id));
      const c = await navigator.credentials.create({publicKey});
      return {id:c.id, rawId:encode(c.rawId), type:c.type, response:{clientDataJSON:encode(c.response.clientDataJSON), attestationObject:encode(c.response.attestationObject), transports:c.response.getTransports()}, clientExtensionResults:c.getClientExtensionResults()};
    };
    const get = async begin => {
      const publicKey = structuredClone(begin.options.publicKey); publicKey.challenge = decode(publicKey.challenge);
      publicKey.allowCredentials?.forEach(c => c.id = decode(c.id));
      const c = await navigator.credentials.get({publicKey});
      return {id:c.id, rawId:encode(c.rawId), type:c.type, response:{clientDataJSON:encode(c.response.clientDataJSON), authenticatorData:encode(c.response.authenticatorData), signature:encode(c.response.signature), userHandle:c.response.userHandle ? encode(c.response.userHandle) : null}, clientExtensionResults:c.getClientExtensionResults()};
    };
    const signin = async () => {const begin = await ok('/authenticators/authentication/begin', {email:'passkey@example.com'}); const s = await ok('/authenticators/authentication/finish', await get(begin), {'X-Assertion-Token':begin.token}); check(s.Proof.Method === 2, 'WebAuthn proof');};
    const otp = async (url, previous = false) => {
      // Keep a previous-step setup code away from the following boundary;
      // verification still uses production trusted time and fixed skew.
      const elapsed = Date.now() % 30000;
      if (previous && elapsed > 27000) await new Promise(resolve => setTimeout(resolve, 30100 - elapsed));
      return (await request('/__test/otp', {url,previous})).data.code;
    };
    const tamper = async (begin, field) => {
      const response = await get(begin);
      if (field === 'signature') {
        const signature = decode(response.response.signature); signature[signature.length-1] ^= 1;
        response.response.signature = encode(signature);
      } else {
        const data = decode(response.response.authenticatorData); data[32] &= ~(field === 'UP' ? 1 : 4);
        response.response.authenticatorData = encode(data);
      }
      return response;
    };
    window.flow = {check, request, ok, denied, password, create, get, signin, otp, tamper};
    return isSecureContext && !!navigator.credentials;
  })()`);
  async function navigate(path, expected = origin + path) {
    await page('Page.navigate', {url: origin + path});
    await waitFor(async () => {
      try { return await evaluate(`location.href === ${JSON.stringify(expected)} && document.readyState === 'complete'`); }
      catch { return false; }
    }, 'navigation');
    assert(await install(), 'localhost secure context and credentials API');
  }
  const cookies = async () => (await page('Network.getCookies', {urls:[origin]})).cookies.filter(c => c.name === 'session');
  async function cookie() {
    const values = await cookies();
    assert.equal(values.length, 1, 'one committed session cookie');
    const c = values[0];
    assert(c.secure && c.httpOnly && c.sameSite === 'Lax' && c.path === '/', 'cookie attributes');
    assert.equal(await evaluate(`document.cookie.includes('session=')`), false, 'bearer hidden from script');
    return c.value;
  }
  const replaceCookie = async value => page('Network.setCookie', {name:'session', value, url:origin, path:'/', secure:true, httpOnly:true, sameSite:'Lax'});
  const stage = name => stages.push(`PASS ${name}`);

  await navigate('/');
  await runControls({evaluate,cookies,stage});
  await evaluate(`flow.password('passkey@example.com')`);
  const weak = await cookie();
  await evaluate(`(async () => {
    await flow.denied('/authenticators/proof'); await flow.denied('/authenticators/manage');
    window.enroll = await flow.ok('/authenticators/enrollment/begin', {email:'passkey@example.com',password:'a distinct safe password'});
  })()`);
  assert((await cookie()) === weak, 'begin cannot replace password cookie');
  if (publishedRegistration) {
    await evaluate(`(async () => {const begin=enroll; ${publishedRegistration}
      flow.check(response.status===200, 'published registration finish');
      window.registration=credential.toJSON();})()`);
  } else {
    await evaluate(`(async () => {window.registration = await flow.create(enroll); await flow.ok('/authenticators/enrollment/finish', registration, {'X-Enrollment-Token':enroll.token});})()`);
  }
  assert.equal((await cookies()).length, 0, 'registration issues no session');
  await evaluate(`flow.denied('/authenticators/enrollment/finish', registration, {'X-Enrollment-Token':enroll.token})`);
  await evaluate(`flow.signin()`);
  const strong = await cookie();
  await evaluate(`flow.ok('/authenticators/proof')`);
  stage('actual registration/sign-in, safe cookie and strong route');

  await evaluate(`flow.password('passkey@example.com')`);
  const actor = await cookie();
  assert(actor !== strong, 'password sign-in issues a distinct bearer');
  await evaluate(`(async () => {await flow.denied('/authenticators/proof'); window.step = await flow.ok('/authenticators/step-up/begin', {});})()`);
  assert((await cookie()) === actor, 'step-up begin cannot issue proof');
  const restricted = await evaluate(`step.token`);
  await replaceCookie(restricted);
  await evaluate(`(async () => {await flow.denied('/authenticators/proof'); await flow.denied('/authenticators/mfa');})()`);
  await replaceCookie(actor);
  for (const field of ['UP', 'UV', 'signature']) {
    await evaluate(`(async () => {await flow.denied('/authenticators/step-up/finish', await flow.tamper(step, ${JSON.stringify(field)}), {'X-Assertion-Token':step.token});})()`);
    assert((await cookie()) === actor, 'invalid browser response preserves cookie');
    await evaluate(`flow.denied('/authenticators/proof')`);
  }
  await evaluate(`(async () => {
    await flow.denied('/authenticators/authentication/begin', {email:'passkey@example.com',proof:2});
    await flow.denied('/authenticators/step-up/begin', {extra:true});
    await flow.denied('/authenticators/authentication/finish', {}, {'X-Assertion-Token':step.token});
    for (const body of ['{}{}', '[]', '{}'+ ' '.repeat(65536)]) {
      const r = await fetch('/authenticators/authentication/begin', {method:'POST',headers:{'Content-Type':'application/json'},body});
      flow.check(r.status === 403, 'browser body bound');
    }
  })()`);
  assert((await cookie()) === actor, 'body and purpose failures preserve cookie');
  stage('real browser UP/UV/signature rejection, pending isolation and wire bounds');
  await evaluate(`(async () => flow.ok('/authenticators/step-up/finish', await flow.get(step), {'X-Assertion-Token':step.token}))()`);
  const elevated = await cookie();
  assert(elevated !== actor, 'step-up rotates actor');
  await replaceCookie(actor);
  await evaluate(`flow.denied('/authenticators/proof')`);
  await replaceCookie(elevated);
  await evaluate(`flow.ok('/authenticators/proof')`);
  stage('actual WebAuthn step-up and retired actor denial');

  await navigate('/authenticators/manage');
  assert.equal(await evaluate(`document.querySelectorAll('li').length`), 1);
  // A second resident credential for the same subject belongs on another
  // device; a single device can replace its resident record for that handle.
  await page('WebAuthn.setAutomaticPresenceSimulation', {authenticatorId, enabled:false});
  const secondDevice = await page('WebAuthn.addVirtualAuthenticator', {options: {...device, transport:'usb'}});
  await evaluate(`document.querySelector('[data-action="webauthn"]:not([data-id])').click()`);
  await waitFor(async () => evaluate(`document.readyState === 'complete' && document.querySelectorAll('li').length === 2`), 'production add passkey control');
  await page('WebAuthn.setAutomaticPresenceSimulation', {authenticatorId, enabled:true});
  await install();
  await cookie();
  await evaluate(`(async () => {window.list = await flow.ok('/authenticators/factors'); window.proof = await flow.ok('/authenticators/proof'); window.extra = list.find(f => f.id !== proof.Proof.FactorID); flow.check(list.length === 2, 'safe list'); flow.check(list.every(f => !('public_key' in f) && !('credential_id' in f)), 'no secret fields'); window.codes = (await flow.ok('/authenticators/backup/issue', {})).codes; flow.check(codes.length === 8, 'backup set');})()`);
  const managed = await cookie();
  await evaluate(`flow.ok('/authenticators/factors/remove', {target:{kind:extra.kind,id:extra.id,revision:extra.revision}})`);
  assert((await cookie()) !== managed, 'management rotates retained actor');
  await evaluate(`(async () => {window.last = (await flow.ok('/authenticators/factors'))[0]; await flow.denied('/authenticators/factors/remove', {target:{kind:last.kind,id:last.id,revision:last.revision}});})()`);
  stage('production management UI, safe listing, rotation and last-factor denial');

  await evaluate(`(async () => {window.recovery = codes[0]; window.backup = await flow.ok('/authenticators/backup/authentication/begin', {email:'passkey@example.com',password:'a distinct safe password'});})()`);
  const beforeBackup = await cookie();
  await evaluate(`(async () => {const s = await flow.ok('/authenticators/fallback/authentication/finish', {code:recovery}, {'X-Fallback-Token':backup.token}); flow.check(s.Proof.Method === 4, 'actual backup proof'); await flow.ok('/authenticators/mfa'); await flow.denied('/authenticators/proof'); await flow.denied('/authenticators/backup/issue', {});})()`);
  assert((await cookie()) !== beforeBackup, 'backup completion issues a distinct bearer');
  const backupActor = await cookie();
  await evaluate(`(async () => {const retry = await flow.ok('/authenticators/backup/authentication/begin', {email:'passkey@example.com',password:'a distinct safe password'}); await flow.denied('/authenticators/fallback/authentication/finish', {code:recovery}, {'X-Fallback-Token':retry.token});})()`);
  assert((await cookie()) === backupActor, 'consumed-code failure issues no cookie');
  stage('actual backup MFA, one-use rejection and phishing-resistant denial');

  for (const [email, stepUp] of [['totp@example.com',false], ['stepup@example.com',true]]) {
    await evaluate(`(async () => {
      const setup = await flow.ok('/authenticators/totp/setup/begin', {email:${JSON.stringify(email)},password:'a distinct safe password'});
      await flow.ok('/authenticators/totp/setup/finish', {code:await flow.otp(setup.url,true)}, {'X-Fallback-Token':setup.token});
      window.totpURL = setup.url;
    })()`);
    assert.equal((await cookies()).length, 0, 'TOTP setup issues no session');
    await evaluate(`flow.password(${JSON.stringify(email)})`);
    const passwordActor = await cookie();
    await evaluate(`(async () => {
      await flow.denied('/authenticators/mfa');
      const begin = await flow.ok('/authenticators/totp/${stepUp ? 'step-up' : 'authentication'}/begin', {${stepUp ? '' : `email:${JSON.stringify(email)},`}password:'a distinct safe password'});
      window.code = await flow.otp(totpURL);
      const s = await flow.ok('/authenticators/fallback/${stepUp ? 'step-up' : 'authentication'}/finish', {code}, {'X-Fallback-Token':begin.token});
      flow.check(s.Proof.Method === 3, 'actual TOTP proof'); await flow.ok('/authenticators/mfa'); await flow.denied('/authenticators/proof'); await flow.denied('/authenticators/manage');
      const retry = await flow.ok('/authenticators/totp/authentication/begin', {email:${JSON.stringify(email)},password:'a distinct safe password'});
      await flow.denied('/authenticators/fallback/authentication/finish', {code}, {'X-Fallback-Token':retry.token});
    })()`);
    assert((await cookie()) !== passwordActor, 'TOTP completion issues a distinct bearer');
    stage(`actual TOTP ${stepUp ? 'step-up' : 'sign-in'}, replay and strong-policy denial`);
  }
  if (concern === 'account-recovery') {
    await runRecovery({origin, page, evaluate, navigate, cookie, cookies, replaceCookie, stage});
  }
  const credentials = await page('WebAuthn.getCredentials', {authenticatorId});
  const secondCredentials = await page('WebAuthn.getCredentials', {authenticatorId:secondDevice.authenticatorId});
  assert.equal(credentials.credentials.length + secondCredentials.credentials.length, 2, 'real virtual-device registration inventory');
