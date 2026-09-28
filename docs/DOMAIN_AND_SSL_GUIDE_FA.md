<div dir="rtl">

# 🌐 راهنمای اتصال دامنه، ساب‌دامین و راه‌اندازی ریورس پروکسی (Reverse Proxy) با SSL

این راهنما شیوهٔ اتصال دامنه یا ساب‌دامین اختصاصی (مانند `panel.yourdomain.com`) به کنترل‌پنل مستر هه‌واڵ (Hawal)، فعال‌سازی گواهی امنیتی رایگان Let's Encrypt و پیکربندی ریورس پروکسی با **Nginx** یا **Caddy** را شرح می‌دهد.

---

## فهرست مطالب
1. [چرا استفاده از دامنه و ریورس پروکسی توصیه می‌شود؟](#۱-چرا-استفاده-از-دامنه-و-ریورس-پروکسی-توصیه-میشود)
2. [تنظیمات DNS (ثبت رکورد A)](#۲-تنظیمات-dns-ثبت-رکورد-a)
3. [نکات کلودفلر: حالت مستقیم (DNS-Only) در برابر پروکسی ابری](#۳-نکات-کلودفلر-حالت-مستقیم-dns-only-در-برابر-پروکسی-ابری)
4. [روش اول: پیکربندی Nginx همراه با Certbot (استاندارد لینوکس)](#روش-اول-پیکربندی-nginx-همراه-با-certbot-استاندارد-لینوکس)
5. [روش دوم: پیکربندی Caddy (دریافت و تمدید خودکار SSL بدون تنظیم دستی)](#روش-دوم-پیکربندی-caddy-دریافت-و-تمدید-خودکار-ssl-بدون-تنظیم-دستی)
6. [نکات فایروال و امنیت پورت‌ها](#۶-نکات-فایروال-و-امنیت-پورتها)
7. [اتصال امن ایجنت‌ها با آدرس دامنه](#۷-اتصال-امن-ایجنتها-با-آدرس-دامنه)

---

## ۱. چرا استفاده از دامنه و ریورس پروکسی توصیه می‌شود؟

به‌صورت پیش‌فرض، پنل هه‌واڵ روی پروتکل خام HTTP و پورت `9090` اجرا می‌شود. با آن‌که این حالت برای استفادهٔ محلی و آزمایشی کاربردی است، اما در سرورهای ابری عمومی دلایل مهمی برای به‌کارگیری دامنه و ریورس پروکسی وجود دارد:
- **رمزنگاری سرتاسری (TLS 1.3):** محافظت از نشست‌های کاربری، توکن‌های محرمانه نودها و فرامین همگام‌سازی تانل در برابر استراق‌سمع.
- **دسترسی استاندارد وب:** استفاده از پورت پیش‌فرض HTTPS یعنی `443` به جای پورت‌های نامتعارف.
- **سهولت در تغییر سرور:** در صورت جابه‌جایی سرور پنل، تنها با تغییر IP در DNS دسترسی همهٔ سیستم‌ها و مدیران بازیابی می‌شود.
- **پایداری استریم و وب‌سوکت:** مدیریت بهتر نشست‌های فعال، Long-Polling و استریم‌های آماری توسط وب‌سرورهای صنعتی.

---

## ۲. تنظیمات DNS (ثبت رکورد A)

1. وارد پنل ارائه‌دهندهٔ خدمات DNS خود (مانند Cloudflare، ابر آروان، Hetzner DNS و ...) شوید.
2. یک رکورد از نوع **A** ایجاد کنید:
   - **Type (نوع رکورد):** `A`
   - **Name (نام رکورد / زیردامنه):** `panel` (یا هر زیردامنهٔ دلخواه دیگر)
   - **Target / IPv4 (مقصد):** `<IP_سرور_پنل>`
   - **TTL:** روی `Auto` یا کمترین مقدار ممکن (مثلاً ۲ دقیقه) تنظیم کنید.
3. صحت انتشار رکورد را با دستور زیر بررسی کنید:
   ```bash
   dig +short panel.yourdomain.com
   # یا
   ping -c 2 panel.yourdomain.com
   ```

---

## ۳. نکات کلودفلر: حالت مستقیم (DNS-Only) در برابر پروکسی ابری

اگر مدیریت DNS دامنهٔ شما روی Cloudflare قرار دارد:
- **حالت پیشنهادی: ابر خاکستری / DNS-Only ⚪**
  وضعیت پروکسی رکورد ساب‌دامین را روی **DNS only** قرار دهید.
  *علت فنی:* ارتباط مستقیم TCP مانع از ایجاد تداخل در دوره‌های Heartbeat ایجنت‌ها، قطعی‌های زمان‌بندی‌شده (Timeouts) و نوسانات پکت‌های نظارتی پنل می‌شود.
- **استفاده از ابر نارنجی (Proxied 🟠):**
  اگر اصرار به پنهان‌سازی IP سرور پنل پشت CDN دارید:
  - قابلیت **WebSockets** را در مسیر `Cloudflare Dashboard -> Network` حتماً فعال کنید.
  - وضعیت SSL را در مسیر `SSL/TLS -> Overview` روی حالت **Full (Strict)** بگذارید.
  - **نکته بسیار مهم:** کلودفلر تنها پورت‌های استاندارد وب را پروکسی می‌کند. **ترافیک پورت‌های هستهٔ تانل‌ها (مانند پورت ۳۱۰۷) بین ایران و خارج هرگز نباید از CDN کلودفلر عبور کند** و این پورت‌ها باید مستقیماً بین دو سرور باز باشند.

---

## ۴. روش اول: پیکربندی Nginx همراه با Certbot (استاندارد لینوکس)

### گام ۱: نصب Nginx و Certbot
در اوبونتو یا دبیان:
```bash
sudo apt-get update
sudo apt-get install -y nginx certbot python3-certbot-nginx
```

### گام ۲: ایجاد تنظیمات سرور Nginx
فایل `/etc/nginx/sites-available/hawal` را ایجاد و محتوای زیر را در آن قرار دهید:
```nginx
server {
    server_name panel.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:9090;
        proxy_http_version 1.1;

        # پشتیبانی از استریم و وب‌سوکت
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # ارسال هدرهای استاندارد کلاینت
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # تنظیم زمان انقضای کانکشن‌های پایدار
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

فعال‌سازی کانفیگ:
```bash
sudo ln -sf /etc/nginx/sites-available/hawal /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### گام ۳: دریافت گواهی امنیتی Let's Encrypt
```bash
sudo certbot --nginx -d panel.yourdomain.com
```
دستورالعمل تعاملی را دنبال کنید تا تغییر مسیر خودکار به HTTPS فعال گردد.

---

## ۵. روش دوم: پیکربندی Caddy (دریافت و تمدید خودکار SSL بدون تنظیم دستی)

وب‌سرور مدرن Caddy دریافت، اعتبارسنجی و تمدید دوره‌ای گواهی SSL را به‌صورت کاملاً خودکار انجام می‌دهد.

### گام ۱: نصب Caddy
```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install -y caddy
```

### گام ۲: ویرایش فایل Caddyfile
فایل `/etc/caddy/Caddyfile` را باز کرده و خطوط زیر را جایگزین کنید:
```caddy
panel.yourdomain.com {
    reverse_proxy 127.0.0.1:9090
}
```

سپس سرویس را راه‌اندازی مجدد نمایید:
```bash
sudo systemctl restart caddy
```
پنل هه‌واڵ اکنون از طریق نشانی امن `https://panel.yourdomain.com` در دسترس شماست.

---

## ۶. نکات فایروال و امنیت پورت‌ها

پس از برقراری ریورس پروکسی:
1. پورت‌های وب عمومی را باز بگذارید:
   ```bash
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   ```
2. (اختیاری) پورت خام `9090` را ببندید تا تنها اتصالات داخلی سرور (Loopback) مجاز باشند:
   ```bash
   sudo ufw delete allow 9090/tcp 2>/dev/null || true
   ```
3. توجه داشته باشید که پورت‌های هستهٔ تانل‌ها (مانند ۳۱۰۷ یا سایر پورت‌های انتخابی شما) جهت تبادل ترافیک بین نودها باید در فایروال باز باشند.

---

## ۷. اتصال امن ایجنت‌ها با آدرس دامنه

هنگامی که نود جدیدی را در داشبورد تعریف می‌کنید، دستور نصب خودکار به شکل نشانی IP تولید می‌شود. با داشتن دامنه و گواهی امنیتی، می‌توانید دستور نصب را با آدرس دامنه و پروتکل امن HTTPS روی سرورهای مقصد اجرا نمایید:
```bash
curl -fsSL "https://panel.yourdomain.com/install?token=YOUR_NODE_TOKEN&role=kharej&name=Germany" | bash
```
ایجنت نود در این حالت تمام ارتباطات مدیریتی و گزارش وضعیت سلامت خود را در بستر رمزنگاری‌شدهٔ HTTPS با پنل مستر تبادل خواهد کرد.

</div>
