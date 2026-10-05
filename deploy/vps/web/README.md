# Web deployment

Both public hostnames serve the responsive web build. Requests under `/api/`
and `/healthz` are forwarded to the backend bound to `127.0.0.1:8081`.

Deploy the built frontend into:

```text
/home/statistics/tools/mock-ai/web/dist
```

Install and enable the Nginx site:

```sh
sudo install -m 644 /home/statistics/tools/mock-ai/web/nginx.conf \
  /etc/nginx/sites-available/mock-ai
sudo ln -sfn /etc/nginx/sites-available/mock-ai \
  /etc/nginx/sites-enabled/mock-ai
sudo nginx -t
sudo systemctl reload nginx
```

After both DNS A records resolve to the VPS, issue one certificate covering
both hostnames and enable HTTP-to-HTTPS redirects:

```sh
sudo certbot --nginx \
  -d mock.web.chessinsight.uz \
  -d mockai.admin.chessinsight.uz \
  --redirect --agree-tos --non-interactive --register-unsafely-without-email
```

Verify automatic renewal:

```sh
sudo certbot renew --dry-run
```
