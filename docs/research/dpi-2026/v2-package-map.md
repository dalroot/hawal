# نقشهٔ پیاده‌سازی Hawal Core v2

آخرین بازبینی: ۳ سپتامبر ۲۰۲۶

| بسته | مسئولیت | وضعیت |
|---|---|---|
| `carrier` | قرارداد Link/Listener و Registry | پیاده‌سازی پایه |
| `carrier/tcp` | مسیر کنترل TCP | پیاده‌سازی و تست Loopback |
| `secure` | مرز Handshake و Traffic Secrets | قرارداد؛ پیاده‌سازی رمزنگاری باز است |
| `protocol` | مذاکرهٔ Version/Capability بعد از Handshake | پیاده‌سازی |
| `record` | AEAD، Metadata محافظت‌شده، Padding و Limits | Prototype تست‌شده |
| `mux` | صف محدود، اولویت و Flow Control | Primitiveهای پایه پیاده‌سازی |
| `session` | Lifecycle و مجموعهٔ چندمسیر | Prototype تست‌شده |
| `controller` | تشخیص خرابی و انتخاب محافظه‌کارانهٔ مسیر | Prototype تست‌شده |
| `observe` | رخدادهای ساختاریافته و Ring محدود | پیاده‌سازی پایه |
| `policy` | سقف‌های عملیاتی مشترک | پیاده‌سازی پایه |
| `shaping` | Plugin و Budget، بدون Profile حدسی | فقط قرارداد و Guardrail |
| `fault` | خطاهای Typed و قابل طبقه‌بندی | پیاده‌سازی پایه |

## مرز نسخهٔ فعلی

این کد هنوز Core اجرایی نیست. برای تبدیل آن به Binary آزمایشی، موارد زیر باقی
مانده‌اند:

1. انتخاب کتابخانه و Pattern امن پس از ADR و بررسی Dependency؛
2. اتصال Handshake به Record Codec و Negotiation؛
3. Stream state machine کامل، ACK/Window update و Record pump؛
4. Supervisor اتصال، Backoff تصادفی و Circuit breaker؛
5. تست Loss/Reorder/Restart و Capture روی دو Endpoint تحت مالکیت؛
6. Binary جداگانه، Feature flag و Rollback؛
7. فقط پس از عبور Gateها، اتصال اختیاری به Agent و پنل.

این تفکیک عمدی است: زیادکردن تعداد فایل یا خطوط، جای Secure Handshake ممیزی‌شده
و تست شبکهٔ کنترل‌شده را نمی‌گیرد.

