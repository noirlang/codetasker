# CodeTasker — İki Yönlü GitHub TODO & Görev Yönetim Platformu

> Yazılımcıların kod içerisine bıraktığı geçici notları (`// TODO:`, `// FIXME:`, `// BUG:`) otomatik olarak takip edilebilir Kanban panolarına dönüştüren ve bu görevleri çift yönlü senkronize eden akıllı bir SaaS platformu.

---

## Neden CodeTasker? (Projenin Doğuş Amacı)

Yazılım geliştirme süreçlerinde, iş planlama araçları (Jira, Trello, GitHub Issues vb.) ile gerçekte yazılan kodlar arasında her zaman ciddi bir **kopukluk** yaşanır. Yazılımcılar kod yazarken karşılaştıkları küçük eksikleri, düzeltilmesi gereken hataları veya gelecekte yapılması gereken işleri kodun içine anlık olarak `// TODO:` veya `// FIXME:` yorum satırları olarak bırakırlar. 

Çoğu zaman bu yorum satırları kod tabanının derinliklerinde **unutulur, kaybolur** ve teknik borç (technical debt) olarak birikmeye devam eder. Her küçük detay için Jira'da kart açmak yazılımcı için büyük bir zaman kaybı ve bürokratik bir yük yaratır.

CodeTasker, bu iki dünyayı birleştirerek yazılımcının çalışma alışkanlıklarını bozmadan iş takibini otomatikleştirmek için geliştirilmiştir.

---

## Büyük ve Orta Ölçekli Şirketlerde Görev Dağılımı ve Yönetimi

### 1. Büyük Ölçekli Şirketler (Enterprise)
* **Zorluk:** Büyük şirketlerde yüzlerce yazılımcı, onlarca farklı mikro servis ve depo (repository) üzerinde çalışır. Ekipler birbirlerinin kod tabanlarından ve oraya bırakılan geçici işlerden habersizdir. Jira veya benzeri araçlar devasa boyutlara ulaşır, karmaşıklaşır ve hantallaşır.
* **Çözüm:** CodeTasker, şirket genelindeki tüm depolarda kodun içerisine yazılan her bir TODO veya bug notunu tek bir merkezde toplar. Böylece ürün yöneticileri (Product Owners) ve mühendislik direktörleri, kod seviyesindeki gerçek teknik eksiklikleri ve yapılması gereken işleri hiçbir yazılımcıyı darlamadan, otomatik olarak görebilirler.

### 2. Orta Ölçekli Şirketler & Hızlı Büyüyen Ekipler (Scale-up & Mid-Market)
* **Zorluk:** Orta ölçekli firmalarda en önemli kriter **hızdır**. Bürokrasiyi azaltmak, gereksiz toplantılardan kaçınmak ve odaklanmış geliştirme yapmak hayati önem taşır. Ancak büyüyen ekiplerde kimin neyi yapacağı, kodun neresinde hangi eksikliklerin kaldığı takip edilemez hale gelir.
* **Çözüm:** Yazılımcılar iş planlama araçlarında vakit kaybetmek yerine doğrudan kod yazarlar. Kodlarını depoya (GitHub) gönderdikleri (push) an, CodeTasker arka planda değişikliği algılar ve Kanban tahtasını günceller. Görevin durumu kodda değişirse panoda da değişir; panoda değişirse koda Pull Request (PR) olarak geri yansır. Sıfır bürokrasi, maksimum hız.

---

## Temel Özellikler

* **Koddan Panoya Senkronizasyon (Push-to-Sync):** Kodunuzda yaptığınız değişikliklerde yeni bir TODO eklediğinizde veya sildiğinizde, platform bunu anında algılar ve ekibin ortak Kanban panosuna yansıtır.
* **Panodan Koda Senkronizasyon (Task Injection):** Doğrudan tarayıcı panosundan kodun belirli bir satırına yeni bir görev (yorum satırı) ekleyebilirsiniz. CodeTasker bunu arka planda otomatik olarak yeni bir dal (branch) açıp, ilgili satıra yorumu ekleyip bir **Pull Request (PR)** oluşturarak yapar.
* **Gelişmiş Commit & İş Birliği Takibi:** Dal birleştirme (merge), commit geçmişini inceleme ve commits/PR'lara çalışma arkadaşlarını (Co-authors) ekleme özellikleriyle tam entegre çalışır.
* **Güvenli JWT & Rol Yönetimi:** Ekiplerin yetki sınırlarını korumak için rol tabanlı erişim kontrolü sunar.

---

## Lisans

GNU General Public License v3.0 (GPL-3.0) - Detaylar için [LICENSE](LICENSE) dosyasına göz atabilirsiniz.
