# ماتریس آزمایش مسیرهای تحت کنترل

وضعیت: آماده برای زمان تحویل سرورهای آزمایش  
اصل ایمنی: فقط IP و سرور متعلق به آزمایش؛ بدون spoof ثالث، اسکن عمومی یا ایجاد بار روی شبکهٔ دیگران

## هدف

تفکیک علت‌های زیر بدون دست‌زدن به Tunnel اصلی:

- outage/routing؛
- port ACL؛
- TCP handshake filtering؛
- post-payload DPI؛
- RST injection؛
- silent drop؛
- UDP blackhole؛
- MTU/MSS؛
- congestion/throttling؛
- خرابی سرویس مقصد.

## اطلاعات اجباری هر Run

```text
run_id:
start/end UTC:
source node/IP/ASN:
destination node/IP/ASN:
direction:
transport:
port:
payload profile:
client version:
server version:
firewall snapshot hash:
pcap source:
pcap destination:
result:
classification:
confidence:
notes:
```

## اصول طراحی آزمایش

1. Receiver و Capture مقصد پیش از Sender شروع شوند.
2. ساعت هر دو سرور با NTP کنترل شود.
3. در هر Run فقط یک متغیر تغییر کند.
4. پورت کنترل و پورت آزمایش هر دو وجود داشته باشند.
5. Payload و مدت بار میان هسته‌ها یکسان باشد.
6. هر Run حداقل سه‌بار و در بازه‌های زمانی متفاوت تکرار شود.
7. Capture کوتاه و دارای filter دقیق باشد.
8. قبل و بعد هر Run وضعیت Listener و firewall ثبت شود.
9. هیچ Rule سرویس اصلی حین آزمایش تغییر نکند.
10. دادهٔ حساس پیش از اشتراک redaction شود.

## مرحلهٔ صفر: سلامت سرویس

| آزمون | مبدا | مقصد | خروجی لازم |
|---|---|---|---|
| ICMP echo | هر دو جهت | IP peer | loss/RTT |
| TCP connect بدون payload | هر دو جهت | پورت آزمایش | SYN/SYN-ACK/ACK |
| Listener local | همان سرور | localhost | service vs network fault |
| UDP echo کنترل‌شده | هر دو جهت | پورت آزمایش | one-way/two-way visibility |
| Route/MTU | هر دو جهت | peer | hop/PMTU indication |

## مرحلهٔ یک: ماتریس Port/Transport

پورت‌ها باید از قبل روی هر دو firewall مجاز و متعلق به آزمایش باشند.

| خانواده | نمونه | جهت‌ها | معیار |
|---|---|---|---|
| TCP handshake only | بدون app data | IR→OUT و OUT→IR | establishment rate |
| TCP opaque | payload تصادفی کنترل‌شده | هر دو | first-write behavior |
| HTTP واقعی | وب‌سرور واقعی | هر دو | status/latency/reset |
| TLS واقعی | گواهی و server واقعی | هر دو | ClientHello/ServerHello/app data |
| UDP کوچک | datagram زیر MTU | هر دو | delivery/loss |
| QUIC واقعی | implementation استاندارد | هر دو | Initial/Handshake/1-RTT |

## مرحلهٔ دو: مقایسهٔ هسته‌ها

برای هر هسته از یک workload مشترک استفاده می‌شود:

- ۱۰۰ اتصال کوتاه؛
- ۱۰ اتصال متوسط؛
- یک انتقال حجیم محدود؛
- idle period؛
- reconnect؛
- TCP و در صورت پشتیبانی UDP.

| هسته | Wire carrier | نسخه | نتیجهٔ Startup | نتیجهٔ Data | پایداری | Capture hash |
|---|---|---|---|---|---|---|
| Hawal v1 | TCP custom | ثبت شود | — | — | — | — |
| Backhaul | انتخاب‌شده | ثبت شود | — | — | — | — |
| GOST | انتخاب‌شده | ثبت شود | — | — | — | — |
| rathole | TCP/TLS/Noise | ثبت شود | — | — | — | — |
| Paqet | Raw TCP/KCP | ثبت شود | — | — | — | — |

## مرحلهٔ سه: آزمایش Failure Semantics

| مشاهده در Capture | تفسیر اولیه | کنترل لازم |
|---|---|---|
| SYN مبدا هست، مقصد نمی‌بیند | drop مسیر/ACL/DPI | پورت و جهت کنترل |
| مقصد SYN را می‌بیند و SYN-ACK برمی‌گرداند، مبدا نمی‌بیند | drop برگشت | TTL و capture میانی در صورت امکان |
| handshake کامل، اولین payload ناپدید | payload-triggered filter محتمل | payload benign و random مقایسه شود |
| RST فقط یک سمت دیده می‌شود | injection محتمل | TTL/IPID/seq/ack و capture دوطرفه |
| چند packet عبور و سپس سکوت دوطرفه | stateful drop محتمل | flow تازه با source port تازه |
| throughput روی سقف ثابت | policing/throttle محتمل | control flow هم‌زمان |
| فقط UDP شکست | UDP policy/NAT محتمل | UDP echo local و reverse direction |
| فقط packet بزرگ شکست | PMTU/fragment issue محتمل | size sweep محدود |
| مقصد connection refused می‌دهد | سرویس مقصد/listener | local connect روی مقصد |

این طبقه‌بندی حکم قطعی نیست؛ هر مورد باید control داشته باشد.

## مرحلهٔ چهار: Sweep اندازه و زمان

پس از مشاهدهٔ failure و فقط روی پورت آزمایش:

- چند bucket اندازه زیر MTU؛
- first payload delay محدود؛
- idle کوتاه/متوسط؛
- keepalive روشن/خاموش؛
- یک flow در برابر pool کوچک؛
- بار interactive در برابر bulk.

هدف کشف boundary است، نه تولید ترافیک زیاد.

## Capture و دادهٔ موردنیاز

برای هر دو سمت:

- packet timestamp با دقت مناسب؛
- IP/TCP/UDP header؛
- packet length و direction؛
- TCP flags، seq/ack و retransmission؛
- TLS/QUIC phase در صورت قابل مشاهده بودن؛
- process log هم‌زمان؛
- socket counter پیش/پس؛
- firewall counter پیش/پس.

Payload واقعی کاربران Capture نمی‌شود. Workload مصنوعی و کنترل‌شده استفاده می‌شود.

## تحلیل آفلاین

خروجی تحلیل باید حداقل شامل موارد زیر باشد:

- timeline دوطرفه؛
- handshake completion rate؛
- time-to-first-byte؛
- reset/drop location inference؛
- loss/reorder/retransmission؛
- throughput و burst distribution؛
- packet-size histogram؛
- direction sequence؛
- overhead نسبت به payload؛
- تفاوت هسته‌ها با confidence interval.

## Gate تصمیم معماری

- اگر TCP پیش از Payload fail شود: camouflage لایهٔ بالا بی‌اثر است؛ carrier/path بررسی شود.
- اگر پس از نخستین Payload fail شود: wire prefix، TLS/SNI یا classifier اولیه بررسی شود.
- اگر پس از زمان/حجم ثابت fail شود: state budget، lifecycle و throttling بررسی شود.
- اگر فقط UDP fail شود: QUIC پیش‌فرض نباشد و fallback سریع لازم است.
- اگر Paqet تنها مسیر سالم باشد: Raw carrier اولویت آزمایشی می‌گیرد، ولی علت با capture اثبات شود.
- اگر failure ناشی از local listener باشد: هیچ نتیجه‌ای درباره DPI ثبت نشود.

## حفاظت از Tunnel اصلی

- سرور و پورت آزمایش جدا؛
- service unit جدا؛
- CPU/bandwidth cap؛
- زمان پایان خودکار؛
- بدون تغییر route پیش‌فرض؛
- بدون reload firewall اصلی؛
- rollback از پیش آماده؛
- REX/SSH مدیریت روی مسیر دیگری باقی بماند.
