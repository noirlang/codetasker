---
name: codetasker
description: CodeTasker CLI kullanım rehberi. Kod tabanı TODO/FIXME tarama, repo senkronizasyonu, görev yönetimi, terminalden task enjekte edip PR açma ve teknik borç analizi iş akışlarını açıklar.
---

# CodeTasker CLI Skill

CodeTasker, kod tabanındaki teknik borçları (`TODO`, `FIXME`, `BUG`, `HACK`, `NOTE`) otomatik tespit eden, GitHub PR'ları ile koda görev enjekte edebilen ve teknik borç maliyet analizi sunan bir sistemdir. Bu beceri (skill), CodeTasker CLI komutlarının kullanımını kapsar.

---

## 📦 Kurulum ve Çalıştırma

### 1. Derleme ve Kurulum
- **Linux & macOS**:
  ```bash
  ./scripts/install-linux.sh
  # Binary ~/.local/bin/codetasker veya /usr/local/bin/codetasker altına kurulur
  ```
- **Windows (PowerShell)**:
  ```powershell
  .\scripts\install-windows.ps1
  # Binary %LOCALAPPDATA%\CodeTasker\bin\codetasker.exe altına kurulur
  ```

---

## ⚙️ CLI Komutları Referansı

### 1. Kod Tabanı Tarama (`scan` - Offline & Yerel)
Veritabanı veya sunucu gerektirmeden yerel projedeki tüm anotasyonları renkli tablolarla listeler:
```bash
codetasker scan .
codetasker scan . --type FIXME
codetasker scan . --json
```

### 2. Repo Yönetimi & Senkronizasyon (`repo`)
```bash
# Senkronize repoları listele
codetasker repo list

# Repoyu baştan sona tarayıp senkronize et
codetasker repo sync <owner/repo>

# Repo dosya ağacını görüntüle
codetasker repo tree <owner/repo> --branch main

# Repo çalışanları ve yetkilerini listele
codetasker repo collab <owner/repo>
```

### 3. Görev Yönetimi & Koda Görev Enjekte Etme (`task`)
```bash
# Görevleri listele
codetasker task list --repo <owner/repo> --status open

# Terminalden koda TODO/FIXME ekleyip GitHub PR aç
codetasker task inject \
  --repo "owner/repo" \
  --file "path/to/file.go" \
  --line 42 \
  --type "TODO" \
  --note "Açıklama metni" \
  --branch "main"

# Görev durumunu veya atanan kişiyi güncelle
codetasker task update <task-id> --status in_progress --assign username

# Göreve yorum ekle / yorumları oku
codetasker task comment add <task-id> "Yorum mesajı"
codetasker task comment list <task-id>
```

### 4. Teknik Borç Analizi (`debt`)
Git geçmişini, kod karmaşıklığını ve churn oranını analiz ederek maliyet hesaplar:
```bash
codetasker debt analyze . --days 90 --cost 35
```

### 5. Bildirimler (`notify`)
```bash
codetasker notify list --unread
codetasker notify read <notification-id>
codetasker notify read-all
```

### 6. Kimlik Doğrulama & Ayarlar (`auth` / `config`)
```bash
codetasker auth login --token "<token>" --server "https://codetasker.noirlang.tr"
codetasker auth status

codetasker config set default_repo "owner/repo"
codetasker config set default_hourly_cost 35
codetasker config list
```
