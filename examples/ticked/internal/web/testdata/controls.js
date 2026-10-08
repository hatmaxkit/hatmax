// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.


// Real browser requests cross the production ingress, PostgreSQL admission and
// application observer. Fixture reads expose no token, password or authority.
async function runControls({evaluate, cookies, stage}) {
  const events = () => evaluate(`(async () => (await fetch('/__test/security-events', {cache:'no-store'})).json())()`);
  const post = (path, fields) => evaluate(`(async () => {
    const started = performance.now();
    const r = await fetch(${JSON.stringify(path)}, {method:'POST', body:new URLSearchParams(${JSON.stringify(fields)})});
    return {status:r.status,text:await r.text(),url:r.url,cache:r.headers.get('Cache-Control'),elapsed:performance.now()-started};
  })()`);
  assert.equal((await events()).length,0,'no observations from GET or fixture setup');
  const email='registered@example.com';
  const password='a distinct safe password';
  const fields={email,password,confirm_password:password};
  const registration=await evaluate(`(async () => Promise.all([1,2].map(async () => {
    const started=performance.now();
    const r=await fetch('/signup',{method:'POST',body:new URLSearchParams(${JSON.stringify(fields)})});
    return {status:r.status,text:await r.text(),url:r.url,elapsed:performance.now()-started};
  })))()`);
  assert.deepEqual(registration.map(({elapsed,...r})=>r),[registration[0],registration[0]].map(({elapsed,...r})=>r),'new and duplicate registration have equal navigation');
  assert(registration.every(r=>r.status===200 && r.url.endsWith('/signin') && r.elapsed>=5900),'finite neutral registration deadline');
  assert.equal((await cookies()).length,0,'registration never issues a session');
  let observed=await events();
  assert.equal(observed.length,2,'one terminal observation per registration request');
  assert.deepEqual(observed.map(e=>e.outcome).sort(),['committed','policy_rejected'],'actual uniqueness race provenance');
  const short=await post('/signup',{email:'candidate@example.com',password:'short',confirm_password:'short'});
  assert.equal(short.status,400,'safe candidate policy feedback');
  assert(short.text.includes('Password is too short') && !short.text.includes('candidate@example.com'),'feedback reveals candidate policy only');
  const before=(await events()).length;
  for(const [path,body] of [
    ['/signin','email=x&email=y&password=z'],
    ['/signin?email=x','email=x&password=z'],
    ['/signin','email=x&password='+ 'x'.repeat(17000)],
  ]) {
    const r=await evaluate(`(async () => {const r=await fetch(${JSON.stringify(path)},{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:${JSON.stringify(body)}});return {status:r.status,text:await r.text()}})()`);
    assert.equal(r.status,400,'malformed browser form rejected before account work');
    assert(!r.text.includes('@example.com'),'structural refusal is redacted');
  }
  assert.equal((await events()).length,before,'transport refusal emits no core event');
  const guarding=await evaluate(`(async () => {
    const attempt=()=>fetch('/signin',{method:'POST',body:new URLSearchParams({email:'capacity-missing@example.com',password:'a distinct safe password'})});
    const initial=(await (await fetch('/__test/security-events')).json()).length;
    const pending=[attempt(),attempt()];
    const until=performance.now()+3000;
    while ((await (await fetch('/__test/ingress-active')).json())!==2) {
      flow.check(performance.now()<until,'owned ingress admission');
      await new Promise(resolve=>setTimeout(resolve,10));
    }
    // Both admitted operations must finish core before measuring the refused
    // third request; their six-second acknowledgment still holds active slots.
    let events;
    do {
      events=await (await fetch('/__test/security-events')).json();
      flow.check(performance.now()<until,'owned terminal observations');
      if(events.length!==initial+2) await new Promise(resolve=>setTimeout(resolve,10));
    } while(events.length!==initial+2);
    const r=await attempt();
    const after=await (await fetch('/__test/security-events')).json();
    const replies=await Promise.all(pending);
    await Promise.all(replies.map(r=>r.text()));
    return {status:r.status,text:await r.text(),before:events.length,after:after.length,replies:replies.map(r=>r.status)};
  })()`);
  assert.equal(guarding.status,429,'active capacity rejects browser work without queue');
  assert.equal(guarding.text,'Request unavailable\n');
  assert.equal(guarding.after,guarding.before,'guard denial cannot invent a core event');
  assert.deepEqual(guarding.replies,[403,403],'admitted requests finish neutral waits');
  assert.equal(await evaluate(`(async () => (await fetch('/__test/ingress-active')).json())()`),0,'acknowledgment releases active capacity');
  const denied=[];
  for(const fields of [
    {email:'missing-browser@example.com',password},
    {email:'budget@example.com',password:'wrong browser password'},
    {email:'budget@example.com',password},
  ]) {
    const r=await post('/signin',fields);
    assert.equal(r.status,403,'neutral password denial');
    assert.equal(r.cache,'no-store');
    assert(r.elapsed>=5900 && r.elapsed<10000,'bounded six-second fixture acknowledgment');
    denied.push(r.text);
    assert.equal((await cookies()).length,0,'denial cannot issue a browser cookie');
  }
  assert(denied.every(text=>text===denied[0] && text==='Authentication unavailable\n'),'missing, wrong and exhausted outcomes share one public response');
  observed=await events();
  assert.deepEqual(observed.slice(-3).map(e=>[e.operation,e.outcome]),[
    ['signin','missing'],['signin','invalid_proof'],['signin','attempts_exhausted'],
  ],'observer retains classified denial behind neutral transport');
  assert(observed.every(e=>!('proof' in e)),'registration and denied passwords assert no proof');
  stage('actual browser registration race, safe policy, shared admission, neutral deadlines and redacted observations');
}
