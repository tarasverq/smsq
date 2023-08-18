Android app forwarding SMS messages to Telegram bot
===================================================

[![license: GPL v3](https://img.shields.io/badge/license-GPL_v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![github pages](https://img.shields.io/badge/github-pages-blue.svg)](https://smsq.me)

It is just a handy thing when you work on your laptop.
You probably don't want to pick up your phone every time you need to enter an OTP code.

## This is parametrized fork of original smsq by Igor Mikushkin

[Original repository](https://github.com/igrmk/smsq)

Since Igor's backend doesn't seem to work anymore, I decided to set up my own.

This fork allows you to set up your own backend and telegram bot.

![image](https://github.com/tarasverq/smsq/assets/8226275/9d411f25-f668-4f14-a33e-df06a3882346)

### Backend installation (optional)

For convenience you can use docker image `tarasverq/smsq-backend`

Just run it with command <br/>
`docker run --name smsq -d --env=DOMAIN=<DOMAIN_HERE> --env=BOT_TOKEN=<BOT_TOKEN_HERE> --env=ADMIN_ID=<YOUR_TG_ID_HERE> -p <PORT_HERE>:80 tarasverq/smsq-backend:latest`

<br/>

| Variable name   | Description                                                                               |
|-----------------|-------------------------------------------------------------------------------------------|
| DOMAIN_HERE     | Domain with port for tg webhook. Webhook can be set up only on ports 80, 88, 443 or 8443  |
| BOT_TOKEN_HERE  | Your bot token obtained from [@BotFather](https://t.me/BotFather)                         |
| YOUR_TG_ID_HERE | Your TG ID. Easiest way to get it: [@username_to_id_bot](https://t.me/username_to_id_bot) |
| PORT_HERE       | Host Machine port                                                                         |


Example:<br/>
`
docker run --name smsq -d --env=DOMAIN=domain.com:443 --env=BOT_TOKEN=6531881811:AAF111e3coTifgug03-MdxN2tUEh7kp4Sm4 --env=ADMIN_ID=123321 -p 8888:80 tarasverq/smsq-backend:latest
`

**IMPORTANT:** Don't forget to send `/start` message to your bot before starting up docker image

In case you want to use https (recommended) set up nginx reverse proxy with letsencrypt certs.
<br/>[Manual](https://gist.github.com/gmolveau/5e5b0bd2773100d85d9302d0fa96632d)

<details>
  <summary>Example nginx conf matching with ports from command above.</summary>

```
server {
    server_name   domain.com;
    location / {
        proxy_pass         http://127.0.0.1:8888;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection keep-alive;
        proxy_set_header   Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }

    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/domain.com/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/domain.com/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot
}
```
</details>

Installation client
------------
Just install mobile app from [releases](https://github.com/tarasverq/smsq/releases) 

Usage
-----

![image](https://github.com/tarasverq/smsq/assets/8226275/9d411f25-f668-4f14-a33e-df06a3882346)

1. Install Android app
2. Open the app
3. Put your bot name to first text field. e.g. `my_sms_bot` 
4. Put your backend url **with ending slash**.  e.g. `https://domain.com/` 
5. start forwarding, connect Telegram
6. Now you receive your SMS messages in this bot!

### In case you don't want to set up backend you can use backend deployed by me.

Bot: `sms_j_bot`<br/>
Backend URL: `https://smsq.jora.wtf/`

Privacy policy
--------------
__[PRIVACY POLICY](PRIVACY.md)__

Thanks to
---------
[![JetBrains](svg/jetbrains.svg)](https://www.jetbrains.com/?from=smsq)
