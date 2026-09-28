# سانسور شبکه و DPI در ۲۰۲۶: مبنای پژوهشی Hawal Core v2

**مخاطب:** توسعه‌دهندگان و نگه‌داران Hawal  
**تاریخ برش پژوهش:** ۳ سپتامبر ۲۰۲۶  
**جغرافیا:** ایران، چین و روسیه  
**بازهٔ اصلی منابع:** ۲۰۲۳ تا ۲۰۲۶؛ منابع قدیمی‌تر فقط برای سازوکارهای پایه  
**هدف تصمیم:** تعریف مدل تهدید، برنامهٔ اندازه‌گیری و الزامات معماری نسل بعدی Hawal Core

## پاسخ اجرایی

یک «پروتکل مخفی ثابت» راه‌حل پایدار نیست. شواهد دانشگاهی نشان می‌دهد سانسورگرهای مدرن می‌توانند بدون رمزگشایی محتوا از ترکیب بایت‌های آغازین، آنتروپی، اندازه و جهت بسته‌ها، Burst، Round Trip، SNI، نسخهٔ QUIC، رفتار Congestion Control و پاسخ سرور به Probe استفاده کنند. بعضی سامانه‌ها پس از تشخیص، RST تزریق می‌کنند؛ بعضی Payload را حذف یا Packetها را Drop می‌کنند؛ برخی Throttle و برخی IP یا Protocol Allowlist اعمال می‌کنند.

نتیجه برای Hawal این است که امنیت، Wire Image و سازگاری شبکه سه مسئلهٔ جدا هستند:

1. **امنیت Session:** احراز هویت، Forward Secrecy، جلوگیری از Replay و حفاظت Metadata.
2. **Carrier:** TCP، TLS/HTTP واقعی، QUIC یا Raw Packet با رفتار استاندارد و قابل‌تعویض.
3. **سازگاری تطبیقی:** تشخیص نوع خرابی و Failover بدون تغییر دستی و بدون قطع Streamهای منطقی.

Paqet روی مسیر فعلی کار می‌کند، در حالی که چند Tunnel مبتنی بر Socket پایدار نیستند. این با فرضیهٔ «فیلتر کامل IP» سازگار نیست؛ اما تا زمان Capture هم‌زمان نمی‌توان میان DPI، Port Policy، TCP State، UDP Policy، MTU یا اختلال کیفیت مسیر حکم قطعی داد.

## روش و معیار اطمینان

منابع به ترتیب زیر وزن داده شده‌اند:

1. مقالهٔ داوری‌شده و Dataset/Artifact قابل بازتولید؛
2. استاندارد IETF و مستندات رسمی پروژه؛
3. دادهٔ اندازه‌گیری OONI، IODA، Censored Planet و IRBlock؛
4. گزارش فنی مستقل با روش شفاف؛
5. گزارش خبری یا تجربهٔ کاربر، فقط برای کشف فرضیه.

درجهٔ اطمینان:

- **بالا:** منبع اولیه، روش روشن و تکرار یا Artifact دارد.
- **متوسط:** دادهٔ معتبر دارد ولی پوشش مکانی/زمانی یا مشاهدهٔ داخل کشور محدود است.
- **پایین:** گزارش عملیاتی یا استنباطی که هنوز مستقل تأیید نشده است.

وجود قابلیت در Release Note یک نرم‌افزار، اثربخشی آن را در ایران اثبات نمی‌کند.

## مدل لایه‌ای سانسورگر

### لایهٔ دسترسی و مسیریابی

- قطع یا کاهش BGP/Reachability؛
- Null-route یا ACL روی Prefix/IP؛
- Allowlist شبکهٔ داخلی یا گروه‌های دسترسی؛
- تفاوت سیاست میان موبایل، ثابت، دیتاسنتر و اپراتورها.

در این لایه هیچ تغییر Payload مشکل را حل نمی‌کند. وقتی فقط ICMP برقرار است، هنوز نمی‌توان سلامت TCP یا UDP را نتیجه گرفت.

### لایهٔ DNS و نام

- پاسخ جعلی یا Poisoning؛
- Drop یا Redirect؛
- Blocklist دامنه، پسوند یا عبارت؛
- مشاهدهٔ SNI یا HTTP Host پس از دورزدن DNS.

رمزکردن DNS به تنهایی کافی نیست، چون IP مقصد، SNI، TLS fingerprint و Flow metadata باقی می‌مانند. ECH در RFC 9849 برای مخفی‌کردن ClientHello داخلی و SNI طراحی شده، اما خود RFC تأکید می‌کند DNS و IP همچنان کانال افشا هستند.

### لایهٔ TCP/UDP و Stateful Filtering

- تطبیق روی پورت یا جهت آغاز اتصال؛
- بررسی نخستین Packetهای داده؛
- بازسازی TCP segment؛
- Drop متقارن یا نامتقارن؛
- RST/ACK جعلی یا تغییر Packet واقعی؛
- محدودیت Fragment queue یا Packet budget؛
- حذف انتخابی UDP/443 یا QUIC Initial.

### لایهٔ پروتکل و Fingerprint

- Magic bytes و نسخه؛
- TLS ClientHello، SNI، ALPN، ترتیب Extensionها و Record boundaries؛
- QUIC version و اندازهٔ Initial؛
- HTTP method/header/path و UUIDهای قابل مشاهده؛
- رفتار خطا و پاسخ Invalid Input.

### تحلیل آماری ترافیک رمز‌شده

- آنتروپی یا popcount Payload آغازین؛
- نسبت ASCII قابل چاپ و موقعیت آن؛
- توالی اندازه و جهت Packet؛
- تعداد Burst و Round Trip؛
- نسبت Uplink/Downlink؛
- طول عمر اتصال و Keepalive؛
- واکنش به Loss و Congestion؛
- اثر TLS تو‌در‌تو داخل Tunnel.

### تأیید فعال و Reputation

- Replay پیام مشاهده‌شده؛
- Probe با Payloadهای مرزی و ناقص؛
- مقایسهٔ پاسخ با HTTP/TLS واقعی؛
- اسکن پورت و هم‌بستگی چند سرویس روی IP؛
- امتیازدهی IP دیتاسنتری، حجم، تعداد کاربران و اتصال‌های طولانی.

## ایران

### یافته‌های با اطمینان بالا

[مقالهٔ IRBlock در USENIX Security 2025](https://www.usenix.org/system/files/usenixsecurity25-tai.pdf) یک سامانهٔ اندازه‌گیری بیرون‌ازکشور ساخت و DNS، HTTP و UDP را در کل فضای IPv4 ایران بررسی کرد. در دورهٔ مقاله بیش از ۵۰۰ میلیون دامنهٔ پایه آزمایش شد؛ ۶٫۸ میلیون IP در معرض DNS poisoning یا HTTP injection و ۵٫۴ میلیون IP در معرض اخلال UDP مشاهده شدند. [Artifact پژوهش در Zenodo](https://zenodo.org/records/15572895) منتشر شده است.

IRBlock رفتارهای زیر را گزارش می‌کند:

- DNS poisoning؛
- تزریق صفحهٔ مسدودسازی HTTP؛
- تزریق `RST+ACK` برای HTTPS؛
- Drop خاموش UDP؛
- فیلتر HTTP در همهٔ پورت‌های TCP آزمایش‌شده، نه فقط پورت پیش‌فرض؛
- رفتار چند Injector و تفاوت سیاست میان شبکه‌ها؛
- شواهد قوی از جزء in-path که می‌تواند Packet را Drop کند؛
- شباهت‌هایی با طراحی GFW چین، اما پیاده‌سازی پراکنده‌تر و انتخابی‌تر.

روش بیرون‌ازکشور IRBlock یک محدودیت اساسی دارد: سیاست‌هایی که فقط با ترافیک آغازشده از داخل ایران فعال می‌شوند دیده نمی‌شوند. پوشش موبایل نیز به علت CGNAT و رد ترافیک ورودی، کف برآورد است.

[پژوهش FOCI 2020 درباره Protocol Whitelister ایران](https://www.usenix.org/conference/foci20/presentation/bock) نشان داد سامانه‌ای جدا از فیلتر عادی، نخستین دو Packet داده را برای Fingerprintهای DNS، HTTP و HTTPS بررسی می‌کرد. این مطالعه تاریخی است و نباید قواعد دقیق سال ۲۰۲۰ را قانون امروز دانست؛ ارزش آن در اثبات معماری چندلایه و محدودیت‌های reassembly/state آن زمان است.

[دادهٔ OONI در ۲۰۲۵](https://explorer.ooni.org/findings/2025-iran-blocked-whatsapp-amid-war-with-israel) نشان داد در برخی شبکه‌های ایران TCP به مقصد WhatsApp برقرار می‌شد، اما TLS پس از ClientHello Timeout می‌داد؛ این با مداخلهٔ TLS/SNI سازگار است. این نتیجه مختص شبکه‌ها و بازهٔ آزمایش است.

### شواهد جدیدتر با اطمینان متوسط

تحلیل‌های ۲۰۲۵ و ۲۰۲۶ از خاموشی‌های ایران از ترکیب BGP، Telescope، OONI و Cloudflare Radar استفاده کرده‌اند و از کاهش شدید دسترسی جهانی، فهرست‌سفید، Throttling و Protocol blocking گزارش می‌دهند. بخشی از این کارها Preprint هستند و باید جدا از IRBlock داوری‌شده نگه داشته شوند.

### پیام مستقیم برای Hawal

- کارکردن Ping فقط Reachability نوع ICMP را ثابت می‌کند.
- کارکردن Paqet و شکست Tunnelهای Socket-based فرضیهٔ فیلتر سطح Flow/Protocol را تقویت می‌کند، نه اثبات.
- UDP را نباید Carrier پیش‌فرض فرض کرد؛ IRBlock اخلال گستردهٔ UDP را مستند کرده است.
- Port hopping بدون تغییر Wire Image ممکن است بی‌اثر باشد، چون HTTP filtering روی پورت‌های متعدد دیده شده است.
- آزمایش باید برای هر ISP و هر جهت جدا ثبت شود.

## چین

### تحلیل ترافیک کاملاً رمز‌شده

[پژوهش USENIX Security 2023](https://www.usenix.org/system/files/usenixsecurity23-wu-mingshi.pdf) نشان داد GFW می‌تواند «looks-like-nothing» را بدون رمزگشایی مستقیم شناسایی کند. قواعد استنباط‌شده شامل fingerprint پروتکل‌های رایج، popcount، نسبت و محل ASCII قابل چاپ و ساختار اولین TCP payload بود. پژوهش همچنین نشان داد اعمال این قاعده محدود به بخشی از Prefixهای دیتاسنتری و بخشی از Flowها بوده؛ احتمالاً برای کاهش false positive و هزینه.

این یافته دو نتیجه دارد:

1. Ciphertext تصادفی به خودی خود Cover Traffic نیست.
2. افزودن Prefix یا دست‌کاری آماری می‌تواند موقتاً یک Rule را دور بزند، اما مقاله آن را stopgap می‌داند؛ سانسورگر می‌تواند ناحیهٔ دیگری از Flow را بررسی یا Active Probe اجرا کند.

### Active Probing

[مطالعهٔ IMC 2020 درباره Shadowsocks](https://gfw.report/publications/imc20/en/) مدل دومرحله‌ای را مستند کرد: شناسایی اولیه با طول/آنتروپی و سپس Probeهای مختلف، شامل Replay و Payloadهای با طول مرزی. پاسخ ثابت، Timeout غیرعادی یا Response length تکراری می‌تواند سرور را تأیید کند.

[HTTPT](https://www.usenix.org/system/files/foci20-paper-frolov.pdf) نشان می‌دهد «سکوت برای همهٔ Probeها» نیز غیرعادی است؛ بیش از ۹۴ درصد سرورهای بررسی‌شده به دست‌کم یک پروتکل رایج پاسخ می‌دادند. دفاع بهتر، قرارگرفتن پشت پیاده‌سازی واقعی و متنوع HTTPS است، نه تقلید دستی nginx.

### وب، DNS و Reassembly

[GFWatch](https://www.usenix.org/system/files/sec21-hoang.pdf) و [GFWeb](https://www.usenix.org/conference/usenixsecurity24/presentation/hoang) نشان می‌دهند GFW مجموعه‌ای چندلایه از DNS، HTTP، HTTPS و TCP/IP blocking دارد. GFWeb طی ۲۰ ماه ۱٫۰۲ میلیارد دامنه را آزمایش کرد و نشان داد GFW برخی ضعف‌های قدیمی، از جمله overblocking و ناتوانی reassembly در بعضی Fragmentها، را اصلاح کرده است. بنابراین موفقیت تاریخی Fragment نباید به امروز تعمیم داده شود.

### QUIC

[پژوهش USENIX Security 2025](https://www.usenix.org/conference/usenixsecurity25/presentation/zohaib) نشان داد GFW از آوریل ۲۰۲۴ QUIC Initial را در مقیاس ملی رمزگشایی و SNI را فیلتر می‌کند. Blocklist QUIC با DNS/TLS/HTTP یکسان نیست. این نتیجه رد می‌کند که «QUIC چون رمز است دیده نمی‌شود».

QUIC همچنان مزایای معماری مانند Stream multiplexing، Connection ID، Migration و Datagram دارد؛ ولی این‌ها ویژگی Transport هستند، نه تضمین ضدسانسور.

### TLS تو‌در‌تو

[پژوهش USENIX Security 2024](https://www.usenix.org/conference/usenixsecurity24/presentation/xue-fingerprinting) نشان داد Handshake TLS داخل Proxy رمز‌شده می‌تواند از روی Burst و Round Trip fingerprint شود. Padding تصادفی و Encapsulation بیشتر لزوماً این اثر را از بین نمی‌برد. Multiplex امیدبخش‌تر است، ولی اگر فقط چند Byte اضافه کند و ساختار Burst باقی بماند کافی نیست.

## روسیه

### معماری TSPU

[مقالهٔ IMC 2022 درباره TSPU](https://censoredplanet.org/papers/tspu-imc22.pdf) این سامانه را به‌عنوان تجهیزات in-path، نزدیک کاربران و تحت کنترل مرکزی Roskomnadzor توصیف می‌کند. مطالعه با vantage point داخل روسیه و آزمون بیرونی، شش رفتار را به سه Trigger اصلی نسبت داد:

- SNI-based؛
- IP-based؛
- QUIC-based.

رفتارهای مشاهده‌شده شامل تغییر Packet به `RST/ACK`، اجازهٔ چند Packet و سپس Drop متقارن، Throttling بسیار شدید، Drop فوری Flow، Block مبتنی بر IP و حذف QUIC بود. سانسور عمدتاً با اتصال آغازشده از داخل روسیه فعال می‌شد و Statefulness داشت.

TSPU نزدیک کاربر قرار دارد؛ این باعث مقیاس افقی و کنترل مرکزی می‌شود، ولی بودجهٔ محاسباتی هر Box را محدود می‌کند. این trade-off مهم است: Strategy ممکن است روی یک ISP یا Load خاص موفق باشد ولی تضمین عمومی ندارد.

### تحولات ۲۰۲۴ تا ۲۰۲۶

[OONI در مسدودسازی Discord روسیه](https://explorer.ooni.org/findings/2025-russia-blocked-discord) Timeout پس از نخستین Write در TLS handshake را گزارش کرده است. گزارش‌های OONI و RKS از گسترش blocking پروتکل‌های VPN و اعمال هماهنگ‌تر سیاست‌ها خبر می‌دهند. این شواهد عملیاتی تازه‌تر از مقالهٔ ۲۰۲۲ هستند، اما جزئیات classifierهای ۲۰۲۶ به اندازهٔ مطالعهٔ TSPU داوری‌شده قابل مشاهده نیست.

در نتیجه، ادعای دقیق درباره اینکه «TSPU در ۲۰۲۶ از فلان مدل ML یا فلان Vendor استفاده می‌کند» بدون Capture یا سند اولیه مجاز نیست.

## مقایسهٔ سه کشور

| قابلیت مشاهده‌شده | ایران | چین | روسیه |
|---|---|---|---|
| DNS manipulation | قوی | قوی | مشاهده‌شده/وابسته به ISP |
| HTTP Host filtering | قوی و چندپورت | قوی | موجود |
| TLS/SNI interference | مشاهده‌شده | گسترده | گسترده و دقیق |
| RST injection/modification | مشاهده‌شده | شناخته‌شده | چند رفتار مستند |
| Silent packet drop | TCP/UDP | انتخابی | مستند |
| UDP/QUIC targeting | اخلال گسترده UDP | decrypt و SNI روی QUIC | fingerprint نسخه/اندازه در مطالعه ۲۰۲۲ |
| Fully-encrypted classifier | گزارش و قرائن | دانشگاهی و قوی | blocking پروتکل، جزئیات جدید محدود |
| Active probing | احتمال/سابقه محدودتر | قوی و مستند | قابلیت عملی، جزئیات کمتر منتشرشده |
| Throttling | گزارش‌شده | ممکن | مستند و هدفمند |
| Allowlist/shutdown | دوره‌ای و مهم | کنترل مرزی پایدار | حرکت به محدودیت گسترده‌تر، شواهد ناهمگون |

این جدول به معنی یکسان‌بودن زمان، پوشش یا فناوری نیست. «مشاهده‌شده» ممکن است مربوط به یک بازه یا subset شبکه‌ها باشد.

## درس‌های فنی از هسته‌های موجود

### Backhaul

Backhaul مسئلهٔ Reverse Tunnel پرظرفیت را با Connection Pool، multiplexing و Transportهای TCP/TCPMux/WS/WSS حل می‌کند. نقطهٔ قوت آن عملیات و Performance است؛ امنیت در برابر classifier تطبیقی هدف اصلی طراحی نیست. Pool ثابت، Keepalive دوره‌ای و Sessionهای طولانی می‌توانند سیگنال رفتاری بسازند.

### GOST v3

GOST معماری Listener/Connector و زنجیرهٔ ماژولار با Transportهای متعدد دارد. درس اصلی آن composability است. تنوع Carrier به‌خودی‌خود مقاومت نمی‌سازد؛ هر ترکیب باید Wire Image و رفتار واقعی خود را داشته باشد.

### rathole

rathole از Noise استاندارد برای امنیت Transport پشتیبانی می‌کند. درس اصلی آن جداسازی احراز هویت و encryption از Forwarding است. Noise خام ممکن است همچنان به عنوان ترافیک با آنتروپی بالا طبقه‌بندی شود؛ امنیت رمزنگاری مساوی camouflage نیست.

### Xray جدید

Releaseهای ۲۰۲۶ Xray قابلیت‌هایی مانند Finalmask، header customization، TCP fragment، UDP noise، Sudoku، XHTTP/3، ECH و browser-like HTTP headers اضافه کرده‌اند. این فهرست برای کشف فضای طراحی مفید است، اما Issueهای همان پروژه نشان می‌دهند composition پیچیده می‌تواند crash، fingerprint تازه یا ناسازگاری ایجاد کند. هر قابلیت باید جدا و ترکیبی تست شود.

## ممیزی Hawal Core فعلی

یافته‌های مستقیم از سورس محلی:

- کلید AES-GCM برابر `SHA-256(token)` است؛ KDF یا DH موقت وجود ندارد.
- Nonce برای هر Encrypt تصادفی است، پس Token به صورت plaintext روی Wire نیست.
- تمام Frameها با Magic ثابت `0x48574C31` یا `HWL1` شروع می‌شوند.
- Type، Stream ID، Payload length و Padding length در هدر ۱۲‌بایتی آشکارند.
- Handshake payload ثابت منطقی `HWL_HS:<token>` دارد، هرچند داخل AEAD قرار می‌گیرد.
- ACK دارای Payload ثابت `OK` است، هرچند رمز می‌شود.
- Padding فقط یکنواخت بین صفر تا ۴۸ بایت است.
- Keepalive TCP برابر ۳۰ ثانیه و reconnect برابر ۳ ثانیه ثابت است.
- پاسخ Active Probe یک صفحهٔ ثابت nginx 1.24.0 با طول و متن ثابت است.
- یک Session فعال قبلی با ورود Session جدید بسته می‌شود.
- Stream migration، key update، replay cache و carrier negotiation وجود ندارند.

نتیجه: اصلاح Magic لازم است ولی کافی نیست. ساختار طول، نوع فریم، timing، رفتار Probe و lifecycle نیز fingerprint می‌سازند.

## اصول معماری Hawal Core v2

### اصل ۱: امنیت و استتار مستقل

- استفاده از Noise pattern یا TLS 1.3 معتبر برای key agreement؛
- ephemeral keys و Forward Secrecy؛
- transcript binding و mutual/server authentication؛
- replay window و token کوتاه‌عمر؛
- key update و nonce discipline؛
- عدم اختراع primitive رمزنگاری.

### اصل ۲: Wire Format بدون Metadata آشکار ثابت

- Magic عمومی ممنوع؛
- نوع Frame، Stream ID، Sequence و length واقعی رمز شود؛
- length bucket و record coalescing قابل تنظیم؛
- padding بر پایهٔ Profile و budget باشد، نه uniform random ثابت؛
- نسخه/قابلیت پس از برقراری کانال امن مذاکره شود.

### اصل ۳: Carrier واقعی و قابل تعویض

- Carrier API مستقل از Session و Stream؛
- TCP/raw، TLS واقعی، HTTP و QUIC به صورت Plugin؛
- masquerade دستی TLS/nginx ممنوع؛
- Invalid traffic در Carrier وب به Origin واقعی برسد؛
- هر Plugin آزمون interoperability، fingerprint و failure semantics داشته باشد.

### اصل ۴: Multiplex چندCarrier

- یک Session منطقی روی چند Carrier؛
- جداسازی Interactive و Bulk؛
- flow control و backpressure؛
- جلوگیری از head-of-line blocking؛
- session resume و migration؛
- reconnect با jitter و exponential backoff محدود.

### اصل ۵: استانداردبودن Congestion Behavior

[پژوهش FOCI 2025 درباره Hysteria و TCP-Brutal](https://petsymposium.org/foci/2025/foci-2025-0001.php) نشان می‌دهد نادیده‌گرفتن سیگنال congestion یک classifier ساده می‌سازد. Hawal نباید برای سرعت، رفتار غیرطبیعی دائمی بسازد. CUBIC/BBR/QUIC استاندارد یا کنترل قابل ارزیابی ترجیح دارد؛ FEC باید adaptive و budgeted باشد.

### اصل ۶: کنترل تطبیقی مبتنی بر مشاهده

کنترلر باید علت خرابی را به کلاس‌های زیر نزدیک کند:

- no route / outage؛
- SYN drop؛
- post-handshake drop؛
- RST injection؛
- UDP blackhole؛
- SNI/TLS interference؛
- throttling؛
- MTU/fragmentation؛
- remote service failure.

تعویض Carrier فقط پس از evidence و با hysteresis انجام شود تا flap و fingerprint ناشی از probing ایجاد نکند.

### اصل ۷: Observability امن

- eventهای state transition به جای log token/payload؛
- counters هر Carrier و هر Stream class؛
- RTT/loss/retransmit/handshake phase؛
- امکان Capture محدود با redaction؛
- export یک Bundle بدون secret برای گزارش باگ.

## چیزهایی که نباید به عنوان راه‌حل اصلی انتخاب شوند

- Fragment ثابت یا یک اندازهٔ «جادویی»؛
- Prefix ثابت برای شبیه‌سازی TLS؛
- Padding بدون مدل Burst و timing؛
- تکیهٔ کامل بر پورت ۴۴۳؛
- QUIC صرفاً به دلیل رمزنگاری؛
- اتصال دائمی واحد برای همهٔ کاربران؛
- aggressive congestion control غیر استاندارد؛
- پاسخ جعلی ثابت به Probe؛
- الگوریتمی که فقط روی یک ISP و یک روز تست شده است؛
- live training پرسر‌وصدا علیه شبکهٔ واقعی.

## برنامهٔ پژوهش تجربی روی مسیر تحت کنترل

مرحلهٔ بعد باید observation-first باشد:

1. Capture هم‌زمان دو سرور برای Baseline مستقیم؛
2. تست TCP، UDP و ICMP در دو جهت و چند پورت کنترل‌شده؛
3. مقایسهٔ Paqet، Backhaul، GOST و Hawal با بار یکسان؛
4. ثبت SYN/SYN-ACK، نخستین Payload، RST، retransmit و silence؛
5. تفکیک startup failure از failure پس از انتقال؛
6. آزمایش MTU/MSS و loss بدون تغییر هم‌زمان چند متغیر؛
7. تکرار در چند بازهٔ زمانی؛
8. تولید hypothesis و سپس فقط یک تغییر در هر آزمایش.

جزئیات در `owned-lab-test-matrix.md` نگه‌داری می‌شود.

## محدودیت‌ها و اختلاف‌ها

- معماری سانسور محرمانه است؛ پژوهش‌ها آن را از رفتار black-box استنباط می‌کنند.
- نتایج ایران و روسیه میان ISP، دسترسی ثابت/موبایل و زمان متفاوت است.
- مقالهٔ داوری‌شده با زمان انتشار عقب‌تر از تغییرات ۲۰۲۶ است.
- Preprintهای خاموشی ۲۰۲۶ برای timeline مفیدند، اما هم‌وزن USENIX/IMC نیستند.
- موفقیت Paqet روی دو سرور یک Case Study است، نه نتیجهٔ عمومی.
- False positive برای classifierها مسئلهٔ اساسی است؛ سانسورگر می‌تواند با محدودکردن Rule به دیتاسنترها یا subset جریان‌ها collateral damage را کاهش دهد.
- هیچ Carrier عمومی تضمین دائمی ندارد؛ هدف مهندسی، کاهش fingerprint ثابت و بازیابی سریع و قابل مشاهده است.

## منابع محوری

1. Tai et al., **IRBlock: A Large-Scale Measurement Study of the Great Firewall of Iran**, USENIX Security 2025: https://www.usenix.org/system/files/usenixsecurity25-tai.pdf
2. IRBlock live research/data portal: https://irblock.org/research
3. IRBlock artifact, Zenodo DOI `10.5281/zenodo.15572895`: https://zenodo.org/records/15572895
4. Bock et al., **Detecting and Evading Censorship-in-Depth: Iran’s Protocol Whitelister**, FOCI 2020: https://www.usenix.org/conference/foci20/presentation/bock
5. OONI, **Iran blocked WhatsApp amid war with Israel**, 2025: https://explorer.ooni.org/findings/2025-iran-blocked-whatsapp-amid-war-with-israel
6. Wu et al., **How the GFW Detects and Blocks Fully Encrypted Traffic**, USENIX Security 2023: https://www.usenix.org/system/files/usenixsecurity23-wu-mingshi.pdf
7. Hoang et al., **GFWeb**, USENIX Security 2024: https://www.usenix.org/conference/usenixsecurity24/presentation/hoang
8. Xue et al., **Fingerprinting Obfuscated Proxy Traffic with Encapsulated TLS Handshakes**, USENIX Security 2024: https://www.usenix.org/conference/usenixsecurity24/presentation/xue-fingerprinting
9. Zohaib et al., **SNI-based QUIC Censorship**, USENIX Security 2025: https://www.usenix.org/conference/usenixsecurity25/presentation/zohaib
10. Xue et al., **TSPU: Russia’s Decentralized Censorship System**, IMC 2022: https://censoredplanet.org/papers/tspu-imc22.pdf
11. OONI, **Russia blocked Discord**, 2025: https://explorer.ooni.org/findings/2025-russia-blocked-discord
12. Frolov and Wustrow, **HTTPT: A Probe-Resistant Proxy**, FOCI 2020: https://www.usenix.org/system/files/foci20-paper-frolov.pdf
13. Wang et al., **Is Custom Congestion Control a Bad Idea for Circumvention Tools?**, FOCI 2025: https://petsymposium.org/foci/2025/foci-2025-0001.php
14. Wang et al., **Minos: Dynamic Defense against Traffic Analysis**, USENIX ATC 2025: https://www.usenix.org/conference/atc25/presentation/wang-zihao
15. IETF, **RFC 9849: TLS Encrypted Client Hello**, 2026: https://www.rfc-editor.org/info/rfc9849/
16. IETF, **RFC 9000: QUIC**, 2021: https://www.rfc-editor.org/rfc/rfc9000.html
17. IETF, **RFC 9221: QUIC DATAGRAM**, 2022: https://www.rfc-editor.org/rfc/rfc9221.html
18. IETF, **RFC 9297: HTTP Datagrams**, 2022: https://www.rfc-editor.org/info/rfc9297
19. IETF, **RFC 9484: CONNECT-IP**, 2023: https://www.rfc-editor.org/rfc/rfc9484.html
20. **Noise Protocol Framework**, revision 34: https://noiseprotocol.org/noise.html
21. Backhaul official repository: https://github.com/Musixal/Backhaul
22. GOST core: https://github.com/go-gost/core
23. rathole transport documentation: https://github.com/rathole-org/rathole/blob/main/docs/transport.md
24. Xray-core 2026 releases: https://github.com/XTLS/Xray-core/releases

## معیار توقف این دور پژوهش

جست‌وجوی عمومی در این دور متوقف شد چون تمام بخش‌های تصمیم‌ساز دارای منبع اولیه هستند، تفاوت سطح اطمینان کشورها ثبت شده، تناقض «رمزنگاری/QUIC/Fragment یعنی نامرئی‌بودن» با شواهد مستقیم حل شده، و جست‌وجوی بیشتر عمدتاً منابع تکراری یا گزارش‌های کم‌اعتبارتر برمی‌گرداند. پژوهش باید پس از دریافت Captureهای مسیر واقعی و با پرسش‌های هدفمند ادامه یابد، نه با جست‌وجوی عمومی بیشتر.
