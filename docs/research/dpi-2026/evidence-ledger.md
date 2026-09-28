# دفتر شواهد و ادعاها

آخرین بازبینی: ۳ سپتامبر ۲۰۲۶

این فایل برای جلوگیری از تبدیل «حدس فنی» به «واقعیت پروژه» نگه‌داری می‌شود. شناسهٔ هر ادعا باید در Issue، ADR یا طرح آزمایش مرتبط ذکر شود.

| شناسه | ادعا | نوع شاهد | منبع و تاریخ | اطمینان | محدودیت/تعارض |
|---|---|---|---|---|---|
| IR-01 | ایران DNS poisoning، HTTP injection و اخلال UDP را در مقیاس وسیع اعمال کرده است | مقاله و Artifact | [IRBlock، USENIX Security 2025](https://www.usenix.org/system/files/usenixsecurity25-tai.pdf) | بالا | اندازه‌گیری بیرون‌ازکشور؛ inside-only policy را نمی‌بیند |
| IR-02 | در دورهٔ IRBlock حدود ۶٫۸ میلیون IP در DNS/HTTP و ۵٫۴ میلیون IP در UDP disruption دیده شدند | مقاله و Dataset | [IRBlock Research](https://irblock.org/research)، ۲۰۲۵–۲۰۲۶ | بالا | عدد مربوط به تعریف و بازهٔ همان سامانه است |
| IR-03 | GFI برای HTTPS می‌تواند پس از دیدن Host/SNI، `RST+ACK` تزریق کند | مقاله | IRBlock، ۲۰۲۵ | بالا | شدت و پوشش میان ASها متفاوت است |
| IR-04 | HTTP filter ایران روی پورت‌های متعدد TCP دیده شده، نه فقط ۸۰ | مقاله | IRBlock، ۲۰۲۵ | بالا | درباره همهٔ Payloadها یا همهٔ ISPها حکم نمی‌دهد |
| IR-05 | اخلال UDP ایران می‌تواند silent drop باشد | مقاله | IRBlock، ۲۰۲۵ | بالا | Trigger دقیق تمام موارد عمومی نشده است |
| IR-06 | Protocol Whitelister سال ۲۰۲۰ نخستین دو data packet را با fingerprintهای مجاز بررسی می‌کرد | مقاله | [FOCI 2020](https://www.usenix.org/conference/foci20/presentation/bock) | بالا تاریخی | Ruleهای دقیق ممکن است مدت‌هاست تغییر کرده باشند |
| IR-07 | بعضی شبکه‌های ایران در ۲۰۲۵ TCP را برقرار ولی TLS را پس از ClientHello متوقف کردند | اندازه‌گیری کاربرمحور | [OONI 2025](https://explorer.ooni.org/findings/2025-iran-blocked-whatsapp-amid-war-with-israel) | متوسط-بالا | مختص دامنه‌ها، ISPها و بازهٔ گزارش |
| IR-08 | خاموشی/فهرست‌سفید می‌تواند با باقی‌ماندن BGP رخ دهد | Preprint/داده عمومی | پژوهش‌های خاموشی ۲۰۲۵–۲۰۲۶ و IODA | متوسط | بخشی هنوز داوری نشده است |
| IR-09 | مسیر فعلی دو سرور IP-block کامل نیست چون Ping و Paqet برقرارند | مشاهدهٔ محلی | لاگ و تست سرورهای تحت کنترل، سپتامبر ۲۰۲۶ | متوسط | ICMP/Paqet علت شکست TCP معمولی را مشخص نمی‌کنند |
| IR-10 | شکست Backhaul/GOST/Hawal ناشی از DPI است | فرضیه | تجربهٔ مسیر فعلی | پایین تا زمان Capture | port policy، MTU، firewall محلی و service failure باید حذف شوند |
| CN-01 | GFW ترافیک fully encrypted را با heuristicهای اولین Payload تشخیص داده است | مقاله | [USENIX Security 2023](https://www.usenix.org/system/files/usenixsecurity23-wu-mingshi.pdf) | بالا | Rule استنباط‌شده و مربوط به بازهٔ اندازه‌گیری است |
| CN-02 | سیگنال‌ها شامل popcount، ASCII position/fraction و protocol exemption هستند | مقاله و بازتولید | USENIX Security 2023 | بالا | classifierهای بعدی ممکن است گسترده‌تر باشند |
| CN-03 | GFW برای Shadowsocks از passive trigger سپس active probing استفاده کرده است | مقاله | [GFW Report/IMC 2020](https://gfw.report/publications/imc20/en/) | بالا تاریخی | رفتار امروز می‌تواند تغییر کرده باشد |
| CN-04 | GFW ضعف‌های قدیمی TCP fragmentation/reassembly را اصلاح کرده است | مطالعه طولی | [GFWeb، USENIX Security 2024](https://www.usenix.org/conference/usenixsecurity24/presentation/hoang) | بالا | همهٔ pathها الزاماً یکسان نیستند |
| CN-05 | از آوریل ۲۰۲۴ GFW QUIC Initial را decrypt و SNI را selectively block کرده است | مقاله | [USENIX Security 2025](https://www.usenix.org/conference/usenixsecurity25/presentation/zohaib) | بالا | چین؛ مستقیم به ایران تعمیم داده نمی‌شود |
| CN-06 | blocklistهای DNS/TLS/HTTP/QUIC یکسان نیستند | مقاله | GFWeb و QUIC censorship، ۲۰۲۴–۲۰۲۵ | بالا | snapshot زمانی |
| CN-07 | fully encrypted «شبیه هیچ‌چیز» می‌تواند خودش کلاس مشکوک باشد | مقاله | USENIX Security 2023 | بالا | میزان false positive deployment-specific است |
| RU-01 | TSPU تجهیزات in-path نزدیک کاربران و با کنترل مرکزی است | مقاله | [IMC 2022](https://censoredplanet.org/papers/tspu-imc22.pdf) | بالا | topology ممکن است از ۲۰۲۲ تغییر کرده باشد |
| RU-02 | TSPU triggerهای SNI، IP و QUIC و شش blocking behavior داشته است | مقاله | IMC 2022 | بالا تاریخی | Rule دقیق ۲۰۲۶ معلوم نیست |
| RU-03 | رفتارها شامل RST/ACK، چند Packet سپس drop، throttle و IP block بوده‌اند | مقاله | IMC 2022 | بالا تاریخی | domain/ISP/time dependent |
| RU-04 | TSPU در مطالعهٔ ۲۰۲۲ اتصال آغازشده از داخل را هدف می‌گرفت و stateful بود | مقاله | IMC 2022 | بالا تاریخی | remote-only test کافی نیست |
| RU-05 | Discord در ۲۰۲۴/گزارش ۲۰۲۵ پس از اولین TLS write timeout می‌شد | OONI | [OONI Russia Discord](https://explorer.ooni.org/findings/2025-russia-blocked-discord) | متوسط-بالا | هدف و بازه محدود |
| RU-06 | blocking پروتکل‌های VPN در ۲۰۲۳–۲۰۲۶ افزایش یافته است | OONI/RKS | [RKS censorship review](https://files.rks.global/censorship_review_en.pdf) | متوسط | جزئیات classifier کمتر از مقالهٔ ۲۰۲۲ است |
| GEN-01 | TLS داخلی داخل tunnel از burst/RTT قابل fingerprint است | مقاله و deployment ISP | [USENIX Security 2024](https://www.usenix.org/conference/usenixsecurity24/presentation/xue-fingerprinting) | بالا | classifier پژوهشی؛ deployment حکومتی اثبات نشده |
| GEN-02 | random padding و encapsulation به تنهایی این fingerprint را حذف نمی‌کند | مقاله | USENIX Security 2024 | بالا | profile و overhead اهمیت دارد |
| GEN-03 | multiplexing امیدبخش است ولی burst size/round trip باید واقعاً تغییر کند | مقاله | USENIX Security 2024 | بالا | طرح دفاع کامل هنوز مسئلهٔ پژوهشی است |
| GEN-04 | سکوت کامل سرور در برابر Probe رفتاری غیرعادی است | مقاله/scan | [HTTPT، FOCI 2020](https://www.usenix.org/system/files/foci20-paper-frolov.pdf) | بالا تاریخی | ترکیب سرورهای اینترنت تغییر می‌کند |
| GEN-05 | استفاده از وب‌سرور واقعی از جعل ناقص TLS/HTTP مقاوم‌تر است | طراحی پژوهشی | HTTPT، ۲۰۲۰ | متوسط-بالا | traffic analysis لایه بالاتر باقی می‌ماند |
| GEN-06 | congestion control تهاجمیِ بی‌اعتنا به loss می‌تواند classifier بسازد | مقاله | [FOCI 2025](https://petsymposium.org/foci/2025/foci-2025-0001.php) | بالا | case study روی Hysteria/TCP-Brutal |
| GEN-07 | Traffic shaping پویا می‌تواند accuracy برخی fingerprint attackها را کم کند | مقاله | [Minos، USENIX ATC 2025](https://www.usenix.org/conference/atc25/presentation/wang-zihao) | بالا برای محیط مطالعه | programmable switch و workload آن با Hawal متفاوت است |
| GEN-08 | ECH، SNI و ClientHello داخلی را می‌پوشاند ولی IP/DNS و correlation باقی می‌ماند | استاندارد | [RFC 9849، ۲۰۲۶](https://www.rfc-editor.org/info/rfc9849/) | بالا | نیازمند ecosystem و anonymity set معتبر |
| GEN-09 | QUIC از migration و stream multiplexing پشتیبانی می‌کند | استاندارد | [RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html) | بالا | مقاومت در برابر DPI را تضمین نمی‌کند |
| GEN-10 | Noise امنیت استاندارد، ephemeral DH و الگوهای احراز هویت فراهم می‌کند | Specification | [Noise Framework](https://noiseprotocol.org/noise.html) | بالا | wire camouflage خارج از هدف Noise است |
| HWL-01 | تمام Frameهای Hawal v1 با `HWL1` آغاز می‌شوند | ممیزی سورس | `core/transport/frame.go`، سپتامبر ۲۰۲۶ | قطعی | مربوط به سورس فعلی repository |
| HWL-02 | Type، Stream ID و lengthها در Header آشکارند | ممیزی سورس | `core/transport/frame.go` | قطعی | — |
| HWL-03 | token plaintext روی Wire نیست؛ Handshake payload با AES-GCM و nonce تصادفی رمز می‌شود | ممیزی سورس | `crypto.go` و `stealth.go` | قطعی | کلید از SHA-256 مستقیم token می‌آید و PFS ندارد |
| HWL-04 | پاسخ Probe ثابت nginx 1.24.0 است | ممیزی سورس | `core/transport/stealth.go` | قطعی | fingerprint فعال ایجاد می‌کند |
| HWL-05 | reconnect سه‌ثانیه‌ای و TCP keepalive سی‌ثانیه‌ای ثابت‌اند | ممیزی سورس | `core/client/client.go` | قطعی | OS ممکن است جزئیات Wire را تغییر دهد |
| XRY-01 | Xray 2026 Finalmask، fragment/noise، Sudoku، XHTTP/3 و browser header profile دارد | Release رسمی | [Xray releases](https://github.com/XTLS/Xray-core/releases) | بالا برای وجود قابلیت | اثربخشی در ایران ثابت نشده است |
| XRY-02 | composition قابلیت‌های جدید می‌تواند regression و fingerprint جدید بسازد | Issueهای رسمی | Xray issues 2026 | متوسط | Issue لزوماً همه نسخه‌ها را درگیر نمی‌کند |

## شکاف‌های باز

| شکاف | اهمیت | روش بستن شکاف |
|---|---:|---|
| علت دقیق عبور Paqet و شکست Socket tunnels در مسیر فعلی | بحرانی | Capture هم‌زمان و ماتریس کنترل‌شده |
| تفاوت رفتار داخل→خارج و خارج→داخل ایران | بحرانی | تست دوطرفه با سرورهای تحت کنترل |
| میزان UDP drop روی ISP و ساعات مختلف | بالا | تکرار زمان‌بندی‌شده، بدون تعمیم یک نمونه |
| Ruleهای فعلی Fragment/reassembly در ایران | بالا | آزمایش کوچک و versioned؛ عدم استفاده از Ruleهای تاریخی به عنوان حقیقت |
| Active probing علیه IPهای Hawal | بالا | honeypot قانونی روی IP خودمان و ثبت Probe؛ بدون پاسخ مخرب |
| اثر traffic shaping بر TLS-in-tunnel | بالا | replay workload یکسان و classifier آفلاین |
| هزینهٔ FEC و multipath | متوسط | benchmark loss/latency/CPU/overhead |
| انتخاب Noise pattern و key lifecycle | بالا | threat review و تست بردار استاندارد |

## قالب افزودن ادعا

```text
ID:
Claim:
Scope (country/ISP/direction/date):
Evidence type:
Primary source:
Corroborating source:
Confidence:
Alternative explanations:
Next test:
```
