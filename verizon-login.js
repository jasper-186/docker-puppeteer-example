const fetch = require('node-fetch');

// ID here is dynamic, and matches up with the immediate response from
// browserless but allows you to track it in third-party systems
module.exports = async ({ page, context, id }) => {
  const { callbackurl, username, password, secretanswer } = context;
  
  await page.goto('https://myvprepay.verizon.com/prepaid/ui/mobile/index.html#/user/landing');
  
  // login page
  await page.type('#IDToken1', username);
  await page.type('#IDToken2', password);
  await page.click('#login-submit');
  await page.waitForNavigation();

  // if #challengequestion exists were at the challenge page
  let challengeForm = await page.$('#challengequestion');
  if(challengeForm){
    await page.type('#IDToken1', secretanswer);
    await page.click('#otherButton');
    await page.waitForNavigation();
  }

  // Get the app part and ship that back to be processed

  const data = await page.$('#app');

  // POST the content to a third-party service
  return fetch(callbackurl, {
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      pageContent: data,
      sessionId: id,
    }),
    method: 'POST',
  });
};