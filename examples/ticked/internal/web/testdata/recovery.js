// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.


// Drive the production forms and actual navigator MFA. Test-only routes read
// captured dispatch and control provider/transport failure, never authentication.
async function runRecovery({origin, page, evaluate, navigate, cookie, cookies, replaceCookie, stage}) {
  const email = 'passkey@example.com';
  const changed = 'a changed browser password 😀';
  const reset = 'a reset browser password 😀';
  const final = 'a final browser password 😀';
  const spent = await evaluate('codes[0]');
  const unused = await evaluate('codes[1]');
  let neutral;
  const install = () => evaluate(`(() => {
    const post = async (path, fields, headers = {}) => {
      const response = await fetch(path, {method:'POST', headers:{'Content-Type':'application/x-www-form-urlencoded', ...headers}, body:new URLSearchParams(fields)});
      return {status:response.status, text:await response.text(), redirect:response.headers.get('HX-Redirect'), cache:response.headers.get('Cache-Control')};
    };
    const mail = async purpose => (await fetch('/__test/recovery-mail?purpose='+purpose, {cache:'no-store'})).json();
    const mode = async failure => {
      const response = await fetch('/__test/mail-mode', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({failure})});
      flow.check(response.status === 204, 'provider fixture');
    };
    window.recovery = {post,mail,mode};
    return true;
  })()`);
  async function open(path) {
    await navigate(path, (origin+path).split('#')[0]);
    await install();
  }
  async function waitFor(check, label) {
    const until = Date.now()+10000;
    while (Date.now()<until) { if (await check()) return; await delay(25); }
    throw Error('recovery browser timeout: '+label);
  }
  async function submit(fields, message) {
    await evaluate(`(() => {
      for (const [name,value] of Object.entries(${JSON.stringify(fields)})) document.querySelector('[name="'+name+'"]').value=value;
      document.querySelector('form').requestSubmit();
    })()`);
    await waitFor(async () => {
      try { return await evaluate(`!document.querySelector('form') && document.body.innerText.includes(${JSON.stringify(message)})`); }
      catch { return false; }
    }, 'committed form response');
    const response = await evaluate('document.body.innerText');
    await installBase();
    return response;
  }
  // A form response replaces the document; reinstall the shared helper through
  // the harness's navigator, then add recovery helpers on its clean fixture page.
  async function installBase() { await open('/'); }
  async function issue(path, address) {
    await open(path);
    const started = Date.now();
    const response = await submit({email:address}, 'If this account is eligible');
    assert(Date.now()-started >= 6000, 'fixed neutral form deadline');
    const expected=path==='/account/mailbox' ? 'If this account is eligible, a verification message will be sent.' : neutral.trim();
    assert.equal(response.trim(),expected,'form acknowledgment remains neutral');
    assert(!response.includes('#token='),'no public/operator bearer');
  }
  async function neutralPost(path, address) {
    const started = Date.now();
    const result = await evaluate(`recovery.post(${JSON.stringify(path)}, {email:${JSON.stringify(address)}})`);
    assert.equal(result.status,202,'neutral acknowledgment');
    assert.equal(result.cache,'no-store');
    assert(Date.now()-started>=6000,'fixed neutral deadline');
    if (neutral === undefined) neutral=result.text;
    else assert.equal(result.text,neutral,'eligibility cannot change acknowledgment');
    assert(!result.text.includes('#token='),'no public/operator bearer');
  }
  const captured = purpose => evaluate(`recovery.mail(${JSON.stringify(purpose)})`);
  const bearer = link => new URLSearchParams(new URL(link).hash.slice(1)).get('token');
  async function denied(path, fields) {
    const result = await evaluate(`recovery.post(${JSON.stringify(path)}, ${JSON.stringify(fields)})`);
    assert.equal(result.status,403,'protected recovery denial');
    assert(!Object.values(fields).some(value => value && result.text.includes(value)),'redacted denial');
  }
  async function password(candidate) {
    const result = await evaluate(`recovery.post('/signin', {email:${JSON.stringify(email)},password:${JSON.stringify(candidate)}})`);
    assert.equal(result.status,200,'ordinary password sign-in');
    assert.equal(result.redirect,'/list-items');
    await cookie();
    await evaluate(`flow.denied('/authenticators/proof')`);
  }
  await install();
  const resetWindowStart=Date.now();
  await neutralPost('/account/password/reset','missing@example.com');
  await neutralPost('/account/password/reset',email);
  assert.equal((await captured('password/reset')).link,'','unverified reset cannot dispatch');
  await evaluate('flow.signin()');
  const verifiedActor=await cookie();
  await issue('/account/mailbox',email);
  const verification=(await captured('mailbox')).link;
  const verifyToken=bearer(verification);
  await open(verification.replace(origin,''));
  assert.equal(await evaluate('location.hash'),'','fragment removed before POST');
  assert((await evaluate('document.querySelector("[name=token]").value'))===verifyToken,'captured token reaches explicit form');
  assert((await cookie())===verifiedActor,'GET cannot revoke actor');
  await evaluate(`flow.ok('/authenticators/proof')`);
  // A distinct browser origin submits the actual simple POST. Its opaque reply
  // and subsequent successful use of the original token establish denial.
  const foreign=new URL(origin); foreign.hostname='127.0.0.1';
  await page('Page.navigate',{url:foreign.origin+'/'});
  await waitFor(async () => evaluate(`location.origin === ${JSON.stringify(foreign.origin)} && document.readyState === 'complete'`),'foreign origin');
  assert.equal(await evaluate(`(async () => (await fetch(${JSON.stringify(origin+'/account/mailbox/confirm')}, {method:'POST',mode:'no-cors',credentials:'include',body:new URLSearchParams({token:${JSON.stringify(verifyToken)}})})).type)()`),'opaque');
  await open(verification.replace(origin,''));
  await replaceCookie(verifyToken);
  await evaluate(`flow.denied('/authenticators/proof')`);
  await replaceCookie(verifiedActor);
  await submit({token:verifyToken},'Mailbox verified');
  assert.equal((await cookies()).length,0,'verification grants no cookie');
  await replaceCookie(verifiedActor);
  await evaluate(`flow.denied('/authenticators/proof')`);
  await evaluate('flow.signin()');
  const fresh=await cookie();
  await denied('/account/mailbox/confirm',{token:verifyToken});
  assert((await cookie())===fresh,'replay preserves current cookie');
  stage('actual mailbox form, GET/fragment safety, cross-origin denial and namespace isolation');

  await password('a distinct safe password');
  const weak=await cookie();
  await denied('/account/password',{password:changed});
  assert((await cookie())===weak,'weaker change proof cannot replace cookie');
  await evaluate('flow.signin()');
  const changeActor=await cookie();
  await open('/account/password');
  assert((await cookie())===changeActor,'change GET has no mutation');
  await submit({password:changed},'Password changed');
  assert.equal((await cookies()).length,0,'change grants no cookie');
  await replaceCookie(changeActor);
  await evaluate(`flow.denied('/authenticators/proof')`);
  await password(changed);
  stage('recent actual MFA authorizes complete Unicode change and ordinary re-entry');

  await evaluate('recovery.mode(true)');
  await neutralPost('/account/password/reset',email);
  const failed=(await captured('password/reset')).link;
  await evaluate('recovery.mode(false)');
  await issue('/account/password/reset',email);
  const link=(await captured('password/reset')).link;
  assert(link!==failed,'failed dispatch cannot revive predecessor');
  await denied('/account/password/reset/confirm',{token:bearer(failed),password:reset});
  await denied('/account/password/reset/confirm',{token:verifyToken,password:reset});
  await evaluate('flow.signin()');
  const resetActor=await cookie();
  await open(link.replace(origin,''));
  assert.equal(await evaluate('location.hash'),'','reset fragment removed');
  assert((await cookie())===resetActor,'reset GET cannot mutate');
  await evaluate(`flow.ok('/authenticators/proof')`);
  await evaluate('recovery.mode(true)');
  await submit({token:bearer(link),password:reset},'Password reset');
  assert.equal((await cookies()).length,0,'reset grants no cookie');
  await evaluate('recovery.mode(false)');
  const notices=(await captured('password/reset')).notices;
  assert.equal(await evaluate(`(async () => (await fetch('/__test/retry-notices',{method:'POST'})).status)()`),204);
  assert((await captured('password/reset')).notices>notices,'failed notice retained for explicit retry');
  await replaceCookie(resetActor);
  await evaluate(`flow.denied('/authenticators/proof')`);
  await denied('/account/password/reset/confirm',{token:bearer(link),password:final});
  await password(reset);
  const passwordActor=await cookie();
  // A consumed code stays consumed; a retained unused verifier still grants
  // actual supported MFA but cannot satisfy the phishing-resistant profile.
  await evaluate(`(async () => {
    const begin=await flow.ok('/authenticators/backup/authentication/begin',{email:${JSON.stringify(email)},password:${JSON.stringify(reset)}});
    await flow.denied('/authenticators/fallback/authentication/finish',{code:${JSON.stringify(spent)}},{'X-Fallback-Token':begin.token});
    const second=await flow.ok('/authenticators/backup/authentication/begin',{email:${JSON.stringify(email)},password:${JSON.stringify(reset)}});
    await flow.ok('/authenticators/fallback/authentication/finish',{code:${JSON.stringify(unused)}},{'X-Fallback-Token':second.token});
    await flow.ok('/authenticators/mfa'); await flow.denied('/authenticators/proof');
  })()`);
  assert((await cookie())!==passwordActor,'actual retained backup code issues MFA');
  stage('one-use reset, failed dispatch/notice retry and retained required factors/backup consumption');

  const before=(await captured('password/reset')).link;
  const weakOperator=await evaluate(`recovery.post('/admin/password-reset',{email:${JSON.stringify(email)}})`);
  assert.notEqual(weakOperator.status,202,'backup proof cannot initiate strong operator reset');
  assert((await captured('password/reset')).link===before,'denied operator cannot issue');
  await evaluate('flow.signin()');
  const operator=await cookie();
  await issue('/admin/password-reset',email);
  assert((await cookie())===operator,'operator initiation grants no access');
  const operatorLink=(await captured('password/reset')).link;
  assert(operatorLink!==before,'qualified operator initiates same mailbox flow');
  await replaceCookie(bearer(operatorLink));
  await evaluate(`flow.denied('/authenticators/proof')`);
  await replaceCookie(operator);
  // Respect the real ingress window after this composite public/operator flow;
  // a rate-limit response must not mask the final one-use storage assertions.
  await delay(Math.max(0,resetWindowStart+61000-Date.now()));
  const lost=await evaluate(`(async () => {
    try { return (await recovery.post('/account/password/reset/confirm',{token:${JSON.stringify(bearer(operatorLink))},password:${JSON.stringify(final)}},{'X-Test-Lose-Response':'yes'})).status !== 200; }
    catch { return true; }
  })()`);
  assert(lost,'committed reset response was lost');
  assert.equal((await captured('password/reset')).lost,1,'one actual committed response dropped');
  assert((await cookie())===operator,'lost response cannot deliver a replacement cookie');
  await evaluate(`flow.denied('/authenticators/proof')`);
  await denied('/account/password/reset/confirm',{token:bearer(operatorLink),password:final});
  await password(final);
  await evaluate('flow.signin()');
  await cookie();
  await evaluate(`flow.ok('/authenticators/proof')`);
  stage('actual role/MFA operator flow, lost committed response, replay denial and required-MFA re-entry');
}
