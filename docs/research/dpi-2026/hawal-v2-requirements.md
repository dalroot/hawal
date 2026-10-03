# الزامات معماری Hawal Core v2

وضعیت: پیش‌نویس پژوهش‌محور، نه مشخصات نهایی پیاده‌سازی  
آخرین بازبینی: ۳ سپتامبر ۲۰۲۶

## هدف محصول

Hawal Core v2 باید یک Session امن و پایدار را روی Carrierهای قابل‌تعویض نگه دارد، علت خرابی مسیر را قابل مشاهده کند و بدون وابستگی به یک Wire fingerprint ثابت Failover انجام دهد.

هدف «نامرئی یا غیرقابل‌تشخیص بودن» نیست. معیار موفقیت، حذف خطاهای شناخته‌شده، کاهش نشانه‌های ثابت، محدودکردن هزینه و بازیابی کنترل‌شده در شبکه‌های متغیر است.

## خارج از دامنهٔ نسخهٔ نخست v2

- اختراع cipher یا KDF؛
- آموزش آنلاین تهاجمی علیه فایروال عمومی؛
- spoof کردن IP متعلق به دیگران؛
- reflection/amplification؛
- domain fronting برخلاف شرایط ارائه‌دهنده؛
- تضمین عملکرد در همه کشورها و ISPها؛
- جایگزینی ناگهانی هستهٔ درحال‌استفاده در سرور اصلی.

## معماری ماژولار

```text
Application Port
      │
Stream/Datagram Adapter
      │
Session + Multiplexer
      │
Scheduler / Migration / Flow Control
      │
Record Protection + Key Lifecycle
      │
Traffic Shaping Policy (optional, budgeted)
      │
Carrier API
 ┌────┼───────────┬────────────┐
 TCP  TLS/HTTP    QUIC         Raw Packet
 └────┴───────────┴────────────┘
      │
Path Observer + Adaptive Controller
```

## ماژول امنیت Session

### الزام‌های MUST

- استفاده از Noise implementation ممیزی‌شده یا TLS 1.3 استاندارد؛
- ephemeral key agreement و Forward Secrecy؛
- احراز هویت دست‌کم سرور؛
- nonce uniqueness و key separation برای هر جهت؛
- transcript binding به نسخه و قابلیت‌ها؛
- replay window محدود و cache با سقف حافظه؛
- key update بر پایهٔ byte/time threshold؛
- صفرکردن secretهای موقت تا حد ممکن؛
- تست رسمی vector، fuzz و property test.

### تصمیم باز

Noise `NK` برای تنظیم ساده شبیه rathole مناسب است؛ `IK/KK` احراز هویت قوی‌تر Client می‌دهد ولی توزیع کلید را پیچیده می‌کند. انتخاب باید با مدل Provisioning نودها هماهنگ شود.

## Record Layer

### MUST

- Magic عمومی و version plaintext نداشته باشد؛
- نوع پیام، Stream ID، Sequence، ACK و length واقعی محافظت شوند؛
- parser دارای سقف length و allocation باشد؛
- قابلیت‌ها بعد از کانال امن مذاکره شوند؛
- unknown version باعث رفتار امن و غیرقابل amplification شود؛
- half-close، cancellation و timeout semantics مشخص باشند.

### SHOULD

- length bucket profile؛
- coalescing چند frame؛
- تقسیم frame بزرگ بر اساس PMTU/Carrier؛
- Padding budget سراسری و per-profile؛
- امکان خاموش‌کردن shaping برای benchmark.

## Multiplexer

### MUST

- چند Carrier برای یک Session؛
- stream-level flow control؛
- connection-level flow control؛
- backpressure واقعی و bounded queue؛
- جلوگیری از starvation؛
- resume token کوتاه‌عمر و bind‌شده به Session؛
- migration idempotent؛
- duplicate suppression؛
- TCP و UDP lifecycle مستقل؛
- عدم قطع Session قبلی صرفاً با ورود Carrier جدید.

### کلاس‌های Stream

- `interactive`: latency-sensitive، queue کوچک؛
- `bulk`: throughput-sensitive؛
- `datagram`: loss-tolerant با expiry؛
- `control`: اولویت بالا و سقف سخت.

## Carrier API

رابط پیشنهادی مفهومی:

```go
type Carrier interface {
    Dial(ctx context.Context, endpoint Endpoint, opts Options) (Link, error)
    Listen(ctx context.Context, bind Bind, opts Options) (Acceptor, error)
    Capabilities() CapabilitySet
    Observe() Metrics
}
```

Carrier نباید منطق token، port mapping یا database داشته باشد.

### Carrier TCP

- baseline ساده برای تست؛
- socket options ثبت‌شده و versioned؛
- keepalive و reconnect دارای jitter؛
- MSS/PMTU قابل مشاهده؛
- رفتار congestion استاندارد.

### Carrier TLS/HTTP

- استفاده از implementation واقعی؛
- certificate lifecycle روشن؛
- ALPN و HTTP behavior سازگار؛
- invalid request به origin واقعی یا پاسخ واقعی stack برسد؛
- عدم ساخت دستی ServerHello یا صفحه nginx؛
- امکان HTTP/2 و HTTP/3 بدون وابستگی Session به نسخه HTTP.

### Carrier QUIC

- QUIC استاندارد و کتابخانهٔ معتبر؛
- Connection ID و migration مطابق RFC؛
- stream و datagram هر دو؛
- congestion استاندارد؛
- fallback روشن هنگام UDP blackhole؛
- Initial/handshake metrics بدون ثبت secret.

### Carrier Raw Packet

- فقط روی سرور تحت کنترل و با root capability محدود؛
- reliability، ACK، retransmission و congestion مستقل اما استانداردرفتار؛
- PMTU و fragmentation کنترل‌شده؛
- FEC تطبیقی با سقف overhead؛
- عدم spoof آدرس ثالث؛
- محافظت در برابر reflection؛
- rate limiting پیش از authentication؛
- سازگاری firewall و cleanup تراکنشی؛
- **ایزوله‌سازی کرنل لینوکس (Kernel RST Suppression):** در هردو سمت کلاینت و سرور قوانین `NOTRACK` و `DROP RST` باید فعال باشند تا پشته TCP سیستم‌عامل در پکت‌های Raw دخالت نکرده و با ارسال بسته‌های Reset ناخواسته، هندشیک تانل را مختل نکند؛
- **دکترین انتخاب پورت (Port Selection Doctrine):** استفاده از بازه پورت‌های Ephemeral (بین ۲۰۰۰۰ تا ۵۵۰۰۰ به صورت رندوم) و اجتناب قاطع از پورت‌های پله‌ای/همسایه برای جلوگیری از شناسایی و بلک‌لیست دامنه‌ای فیلترینگ.

## Traffic Shaping

Shaping یک Plugin اختیاری است، نه بخشی از crypto.

### ورودی Policy

- Packet direction؛
- record size؛
- stream class؛
- elapsed time from handshake؛
- current RTT/loss؛
- overhead budget؛
- carrier constraints.

### خروجی Policy

- bucket size؛
- bounded delay؛
- coalesce/split؛
- dummy budget؛
- burst schedule.

### محدودیت‌ها

- profile ثابت جهانی ممنوع؛
- overhead و latency باید در Telemetry دیده شوند؛
- shaping نباید congestion response را پنهان یا نقض کند؛
- مدل باید آفلاین و روی Captureهای کنترل‌شده ارزیابی شود؛
- **سیاست سنجش تاخیر (Latency Measurement):** پروبینگ مصنوعی دوره‌ای و با فرکانس بالا ممنوع است. سنجش تاخیر و سلامت مسیر باید ترجیحاً به صورت غیرفعال (In-Band RTT روی ارتباطات کنترل یا داده موجود) انجام شود تا از ایجاد پترن رفتاری قابل تشخیص برای DPI جلوگیری شود.

## Adaptive Controller

### ورودی‌ها

- DNS/connect/handshake زمان‌بندی مرحله‌ای؛
- SYN retry و TCP reset؛
- QUIC handshake timeout؛
- loss/reorder/retransmission؛
- throughput و congestion response؛
- remote service health؛
- نتیجهٔ Probe کم‌حجم کنترل‌شده.

### خروجی‌ها

- انتخاب یا تعویض Carrier؛
- تعداد Carrier؛
- migration؛
- FEC budget؛
- shaping profile؛
- circuit breaker؛
- گزارش علت با confidence.

### قواعد ایمنی

- حداقل dwell time؛
- hysteresis برای جلوگیری از flap؛
- rate limit آزمایش؛
- عدم تعویض چند متغیر هم‌زمان؛
- rollback خودکار؛
- حفظ آخرین تنظیم سالم؛
- حالت دستی و kill switch.

## Observability

### Event schema حداقلی

```text
timestamp
node_id
tunnel_id
carrier_id
phase: resolve|connect|secure_handshake|session|stream
event: success|timeout|reset|drop_suspected|migrate|retry
direction
rtt_ms
loss_percent
bytes_in/out
confidence
```

Token، کلید، payload، UUID کاربر و destination حساس نباید در Log پیش‌فرض بیایند.

## تست‌ها و Gate انتشار

### امنیت

- test vector؛
- replay؛
- nonce reuse detection؛
- malformed frame fuzzing؛
- memory/CPU amplification؛
- unauthenticated rate limit؛
- key rotation؛
- downgrade resistance.

### پایداری

- packet loss ۰ تا ۳۰ درصد؛
- reorder و duplication؛
- NAT rebinding؛
- carrier loss و migration؛
- MTUهای متفاوت؛
- restart هر سمت؛
- ۲۴ ساعت soak test؛
- هزاران stream کوتاه و چند stream حجیم.

### Wire analysis

- ثابت‌نبودن prefix میان Sessionها؛
- نبود metadata آشکار؛
- distribution size/timing؛
- TLS-in-tunnel burst test؛
- response to invalid input؛
- مقایسه با baseline و نسخهٔ قبل.

### Release Gate

نسخه تنها وقتی `experimental` می‌شود که:

1. تمام تست‌های crypto و parser پاس شوند؛
2. crash یا unbounded allocation نداشته باشد؛
3. migration در loss و restart تأیید شود؛
4. Capture baseline و گزارش Wire منتشر شود؛
5. نصب کنار v1 و rollback بدون قطع امکان‌پذیر باشد.

عنوان «stable» علاوه بر این‌ها به soak test چندشبکه‌ای و عدم regression در دو Release متوالی نیاز دارد.

## ADRهای لازم پیش از کدنویسی

- ADR-001: انتخاب Noise pattern و library؛
- ADR-002: wire record envelope؛
- ADR-003: Stream ID، sequence و replay window؛
- ADR-004: Carrier interface؛
- ADR-005: migration semantics؛
- ADR-006: congestion/FEC policy؛
- ADR-007: observability و privacy؛
- ADR-008: deployment کنار v1 و rollback؛
- ADR-009: threat-testing ethics؛
- ADR-010: compatibility/version negotiation.

## ترتیب پیاده‌سازی پیشنهادی

1. capture/diagnostic harness؛
2. secure session library؛
3. record layer و fuzz tests؛
4. mux با TCP carrier baseline؛
5. multi-carrier و migration؛
6. observability؛
7. TLS/HTTP carrier واقعی؛
8. raw carrier آزمایشی؛
9. adaptive controller؛
10. shaping profileها پس از داشتن Dataset.

ساخت Shaping پیش از Dataset باعث حدس‌زدن رفتار طبیعی و تولید fingerprint تازه می‌شود.
