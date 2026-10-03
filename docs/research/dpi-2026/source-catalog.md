# فهرست منابع پژوهش

تاریخ دسترسی: ۳ سپتامبر ۲۰۲۶

این فهرست منابع محوری را ثبت می‌کند. «کاربرد» مشخص می‌کند هر منبع برای اثبات چه چیزی استفاده شده و «محدودیت» مانع تعمیم نادرست آن می‌شود.

## مقالات داوری‌شده و Artifactها

| شناسه | عنوان | نویسنده/ناشر | تاریخ | پیوند | کاربرد و محدودیت |
|---|---|---|---|---|---|
| S01 | IRBlock: A Large-Scale Measurement Study of the Great Firewall of Iran | Jonas Tai, Karthik N. Sengottuvelavan, Peter Whiting, Nguyen Phong Hoang؛ USENIX Security | اوت ۲۰۲۵ | [مقاله](https://www.usenix.org/system/files/usenixsecurity25-tai.pdf) | مرجع اصلی ایران؛ remote measurement و محدودیت inside-only/mobile دارد |
| S02 | IRBlock Artifact | همان پژوهشگران؛ Zenodo | ژوئن ۲۰۲۵ | [Dataset و کد، DOI 10.5281/zenodo.15572895](https://zenodo.org/records/15572895) | بازتولید DNS/HTTP/UDP؛ حجم کامل Dataset در این دور دانلود نشد |
| S03 | Detecting and Evading Censorship-in-Depth: Iran’s Protocol Whitelister | Kevin Bock et al.؛ USENIX FOCI | اوت ۲۰۲۰ | [صفحه مقاله](https://www.usenix.org/conference/foci20/presentation/bock) | معماری چندلایه و fingerprint اولیه؛ تاریخی است |
| S04 | How the Great Firewall Detects and Blocks Fully Encrypted Traffic | Mingshi Wu et al.؛ USENIX Security | اوت ۲۰۲۳ | [PDF](https://www.usenix.org/system/files/usenixsecurity23-wu-mingshi.pdf) | مرجع entropy/popcount/ASCII و passive classifier چین |
| S05 | How China Detects and Blocks Shadowsocks | Alice et al./GFW Report؛ ACM IMC | ۲۰۲۰ | [نسخه پژوهش](https://gfw.report/publications/imc20/en/) | passive trigger و active probing؛ تاریخی ولی بنیادی |
| S06 | GFWeb: Measuring the Great Firewall’s Web Censorship at Scale | Nguyen Phong Hoang et al.؛ USENIX Security | اوت ۲۰۲۴ | [صفحه مقاله](https://www.usenix.org/conference/usenixsecurity24/presentation/hoang) | مطالعهٔ ۱٫۰۲ میلیارد دامنه و تحول reassembly؛ بازه داده ۲۰۲۲–۲۰۲۳ |
| S07 | How Great is the Great Firewall? Measuring China’s DNS Censorship | Nguyen Phong Hoang et al.؛ USENIX Security | ۲۰۲۱ | [PDF](https://www.usenix.org/system/files/sec21-hoang.pdf) | مرجع DNS poisoning و GFWatch |
| S08 | Exposing and Circumventing SNI-based QUIC Censorship | Ali Zohaib et al.؛ USENIX Security | اوت ۲۰۲۵ | [صفحه مقاله](https://www.usenix.org/conference/usenixsecurity25/presentation/zohaib) | مرجع decrypt/parse QUIC Initial و blocklist مستقل چین |
| S09 | Fingerprinting Obfuscated Proxy Traffic with Encapsulated TLS Handshakes | Diwen Xue et al.؛ USENIX Security | اوت ۲۰۲۴ | [صفحه مقاله](https://www.usenix.org/conference/usenixsecurity24/presentation/xue-fingerprinting) | TLS-in-tunnel classifier؛ deployment پژوهشی ISP و نه اثبات استفاده دولتی |
| S10 | TSPU: Russia’s Decentralized Censorship System | Diwen Xue et al.؛ ACM IMC | اکتبر ۲۰۲۲ | [PDF](https://censoredplanet.org/papers/tspu-imc22.pdf) | مرجع اصلی معماری و شش رفتار TSPU؛ Ruleهای ۲۰۲۶ ممکن است تغییر کرده باشند |
| S11 | HTTPT: A Probe-Resistant Proxy | Sergey Frolov, Eric Wustrow؛ USENIX FOCI | ۲۰۲۰ | [PDF](https://www.usenix.org/system/files/foci20-paper-frolov.pdf) | active-probe threat و استفاده از HTTPS واقعی؛ تاریخی |
| S12 | OpenVPN is Open to VPN Fingerprinting | Diwen Xue et al.؛ USENIX Security | اوت ۲۰۲۲ | [صفحه مقاله](https://www.usenix.org/conference/usenixsecurity22/presentation/xue-diwen) | مدل passive fingerprint سپس active confirmation |
| S13 | Is Custom Congestion Control a Bad Idea for Circumvention Tools? | Wayne Wang et al.؛ FOCI/PETS | ۲۰۲۵ | [مقاله](https://petsymposium.org/foci/2025/foci-2025-0001.php) | classifier رفتار congestion در Hysteria/TCP-Brutal؛ short paper |
| S14 | Minos: A Lightweight and Dynamic Defense against Traffic Analysis | Zihao Wang et al.؛ USENIX ATC | ژوئیه ۲۰۲۵ | [صفحه مقاله](https://www.usenix.org/conference/atc25/presentation/wang-zihao) | traffic morphing پویا؛ محیط programmable data plane با Hawal متفاوت است |
| S15 | Traffic Shaping for Network Protocols: A Modular Framework | FOCI/PETS | ۲۰۲۵ | [PDF](https://petsymposium.org/foci/2025/foci-2025-0011.pdf) | ایدهٔ shim مستقل از Transport؛ extended abstract و نیازمند ارزیابی بیشتر |

## اندازه‌گیری‌ها و گزارش‌های فنی

| شناسه | عنوان/سامانه | ناشر | تاریخ | پیوند | کاربرد و محدودیت |
|---|---|---|---|---|---|
| S16 | IRBlock live research portal | تیم IRBlock | به‌روزرسانی تا ۲۰۲۶ | [پرتال](https://irblock.org/research) | ادامهٔ Dataset پس از مقاله؛ برخی داده‌های تفصیلی با درخواست پژوهشی ارائه می‌شوند |
| S17 | Iran blocked WhatsApp amid war with Israel | OONI | اوت ۲۰۲۵ | [یافته](https://explorer.ooni.org/findings/2025-iran-blocked-whatsapp-amid-war-with-israel) | TLS timeout پس از ClientHello در subset شبکه‌های ایران |
| S18 | Russia blocked Discord | OONI | ژانویه ۲۰۲۵ | [یافته](https://explorer.ooni.org/findings/2025-russia-blocked-discord) | TLS-level interference؛ محدود به هدف/بازه گزارش |
| S19 | TSPU technical study | Censored Planet | همراه مطالعه IMC 2022 | [نسخهٔ آرشیوی مقاله](https://censoredplanet.org/assets/tspu-imc22.pdf) | نسخهٔ رسمی جایگزین برای دسترسی به شکل‌ها و روش مطالعه |
| S20 | Censorship review / VPN protocol blocking | RKS Global | ۲۰۲۴–۲۰۲۵ | [PDF](https://files.rks.global/censorship_review_en.pdf) | تازگی بیشتر برای روسیه؛ جزئیات classifier محدودتر از IMC |

## استانداردها و مستندات فنی اولیه

| شناسه | عنوان | ناشر | تاریخ | پیوند | کاربرد و محدودیت |
|---|---|---|---|---|---|
| S21 | RFC 9849: TLS Encrypted Client Hello | IETF/RFC Editor | مارس ۲۰۲۶ | [RFC](https://www.rfc-editor.org/info/rfc9849/) | حریم خصوصی ClientHello؛ IP/DNS/correlation را به تنهایی حل نمی‌کند |
| S22 | RFC 9000: QUIC | IETF/RFC Editor | مه ۲۰۲۱ | [RFC](https://www.rfc-editor.org/rfc/rfc9000.html) | migration، stream و transport semantics؛ ضدسانسور بودن را تضمین نمی‌کند |
| S23 | RFC 9221: QUIC DATAGRAM | IETF/RFC Editor | مارس ۲۰۲۲ | [RFC](https://www.rfc-editor.org/rfc/rfc9221.html) | datagram امن داخل QUIC |
| S24 | RFC 9297: HTTP Datagrams | IETF/RFC Editor | اوت ۲۰۲۲ | [RFC](https://www.rfc-editor.org/info/rfc9297) | پایه MASQUE datagram/capsule |
| S25 | RFC 9484: CONNECT-IP | IETF/RFC Editor | اکتبر ۲۰۲۳ | [RFC](https://www.rfc-editor.org/rfc/rfc9484.html) | الگوی استاندارد proxy IP روی HTTP |
| S26 | Noise Protocol Framework، revision 34 | Trevor Perrin/Noise Project | ژوئیه ۲۰۱۸ | [Specification](https://noiseprotocol.org/noise.html) | امنیت handshake؛ camouflage جزو هدف آن نیست |
| S27 | Backhaul | Musixal | دسترسی ۲۰۲۶ | [مخزن رسمی](https://github.com/Musixal/Backhaul) | transport/pool/mux؛ ادعای ضد DPI از وجود ویژگی‌ها نتیجه نمی‌شود |
| S28 | GOST Core | go-gost | دسترسی ۲۰۲۶ | [مخزن رسمی](https://github.com/go-gost/core) | معماری listener/connector ماژولار |
| S29 | rathole Transport | rathole-org | دسترسی ۲۰۲۶ | [مستندات رسمی](https://github.com/rathole-org/rathole/blob/main/docs/transport.md) | TLS/Noise؛ اثر wire باید جدا سنجیده شود |
| S30 | Xray-core releases | XTLS | انتشارهای ۲۰۲۶ | [Releaseهای رسمی](https://github.com/XTLS/Xray-core/releases) | کشف قابلیت‌های Finalmask/XHTTP/ECH؛ نه شاهد موفقیت در ایران |
| S31 | Geneva: Evolving Censorship Evasion at the Transport Layer | Kevin Bock et al.؛ ACM CCS / Black Hat USA | ۲۰۲۰ | [صفحه مقاله و ارائه Black Hat](https://geneva.cs.umd.edu/papers/geneva_ccs19.pdf) | مبنای نظری دور زدن TCB دستگاه‌های DPI با دستکاری فریم‌های خام و پرچم‌های TCP |
| S32 | Insertion, Evasion, and Denial of Service: Eluding Network Intrusion Detection | Thomas H. Ptacek, Timothy N. Newsham | ۱۹۹۸ | [گزارش پژوهشی تاریخی](http://insecure.org/stf/secnet_ids/secnet_ids.html) | تفاوت تحلیل جریان میان فایروال میانی و مقصد نهایی در بازسازی استک TCP |
| S33 | udp2raw-tunnel & FakeTCP | Wang Yu | ۲۰۱۷–۲۰۲۰ | [مخزن رسمی](https://github.com/wangyu-/udp2raw-tunnel) | مبنای کپسوله‌سازی بسته‌های داده درون هدرهای TCP ساختگی با سوکت خام |
| S34 | paqet: transport over raw packets | hanselime | ۲۰۲۴–۲۰۲۶ | [مخزن رسمی](https://github.com/hanselime/paqet) | پیاده‌سازی کاربردی KCP بر بستر فریم‌های خام لایه ۲ اترنت |

## منابعی که عمداً مبنای نتیجه نشدند

- Reddit، Telegram و تجربه‌های بدون Capture: فقط سیگنال کشف.
- Wikipedia: فقط یافتن واژه یا تاریخ اولیه؛ در ادعاهای اصلی استفاده نشد.
- خبرهای تجاری VPN: برای ادعای classifier یا میزان موفقیت استفاده نشد.
- ادعای Vendor/مدل ML فایروال ایران یا روسیه بدون سند اولیه: حذف شد.
- Preprintهای ۲۰۲۶: برای timeline و فرضیه مفیدند، اما هم‌وزن مقاله داوری‌شده نیستند.

## سابقهٔ جست‌وجو و معیار توقف

جست‌وجوها خانواده‌های زیر را پوشش دادند: Iran/GFI/IRBlock، China/GFW/active probing/fully encrypted/QUIC، Russia/TSPU/VPN protocol blocking، TLS-in-tunnel، congestion fingerprint، traffic shaping، probe resistance، Noise، ECH، QUIC و MASQUE. پس از آن جست‌وجوی هدفمند برای شکاف تازگی روسیه، inside/outside ایران و قابلیت‌های Xray 2026 انجام شد.

توقف زمانی انجام شد که ادعاهای تصمیم‌ساز دست‌کم یک منبع اولیه داشتند و نتایج تازه عمدتاً تکراری، خبری یا کم‌روش‌تر بودند. مرحلهٔ بعدی باید بر Capture مسیر واقعی متمرکز باشد.
