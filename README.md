<div align="center">

<img src="frontend/public/logo-kucuk.png" alt="CodeTasker Logo" width="120" />

# CodeTasker — Two-Way GitHub TODO & Task Management Engine

*Automatically convert inline code comments (TODO, FIXME, BUG) into interactive Kanban tasks, and inject them back into your codebase via automated Pull Requests.*

[Website](https://noirlang.tr) | [GitHub Repository](https://github.com/noirlang/codetasker) | [Contributing](CONTRIBUTING.md)

<video src="" width="700" controls></video>

</div>

## Overview

CodeTasker is an intelligent task synchronization platform that bridges the gap between your codebase and project management tools. It automatically scans your synchronized GitHub repositories for annotations like `// TODO:`, `// FIXME:`, `// BUG:`, `// HACK:`, and `// NOTE:`, maps them onto a visual Kanban board, and allows you to create and inject comments back into your code via automated Git branches and Pull Requests.

It is designed for engineering organizations of all sizes to eliminate technical debt tracking overhead and keep tasks perfectly in sync with the actual code.

## Özellikler (Features)

CodeTasker platformunun sunduğu temel özellikler ve ekran görüntüleriyle gerçek kullanım senaryoları aşağıda açıklanmıştır:

### 1. Push-to-Sync (Koddaki Yorumlardan Görev Panosuna)
Kod tabanınızda yer alan satır içi yorumlar (`# TODO:`, `# FIXME:`, `# BUG:`, vb.) otomatik olarak taranır ve interaktif görevlere dönüştürülür. GitHub'a yapılan push işlemleri, panodaki görev durumlarını anında senkronize eder.
* **Görsel Tanımı:** Ruby dosyasındaki (`main.rb`) yorum satırlarının taranarak sağ taraftaki "Tasks" sekmesinde "OPEN" kolonunda listelenmesi.
* **Ekran Görüntüsü:**
  ![Push-to-Sync Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-29-38.png)

### 2. Dashboard ve Senkronize Repolar (Dashboard & Synced Repos)
Kullanıcının sahip olduğu tüm GitHub depolarını yönetebildiği ana kontrol paneli ve CodeTasker ile aktif olarak senkronize edilmiş depoların listelendiği yönetim ekranı. Sağ üst menüden kişisel repolar ve organizasyonlar arasında kolayca geçiş yapılabilir.
* **Görsel Tanımı:** Sahip olunan tüm repoların listelendiği "Dashboard" ekranı ve aktif senkronize repoların (`codetester-test`, `worm-next`) gösterildiği "Synced Repos" arayüzü.
* **Ekran Görüntüleri:**
  ![Dashboard Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-31-02.png)
  ![Synced Repos Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-31-07.png)

### 3. Görev Enjeksiyonu (Task Injection - Panodan Koda)
Uygulama arayüzünden doğrudan hedef koda yeni bir görev (inline comment) ekleyebilirsiniz. CodeTasker dosya uzantısını algılar, dilin yorum satırı sözdizimine uygun olarak yorumu yazar, yeni bir branch açar ve otomatik Pull Request oluşturur.
* **Görsel Tanımı:** Arayüzdeki "Inject TODO" yan paneli kullanılarak `main.rb` dosyasının 5. satırına doğrudan yeni bir `TODO` görevinin ve otomatik PR'ının eklenmesi aşaması.
* **Ekran Görüntüsü:**
  ![Task Injection Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-30-00.png)

### 4. PR Yönetimi ve Doğrudan Birleştirme (PR Management & Direct Merge)
Koda enjekte edilen görevler için açılan Pull Request'ler CodeTasker paneli üzerinden doğrudan yönetilebilir ve birleştirilebilir (merge).
* **Görsel Tanımı:** Sağ paneldeki "Pull Requests" sekmesi altında daha önce koda eklenmiş ve birleştirilerek kapatılmış (`closed`) olan otomatik Pull Request listesi.
* **Ekran Görüntüsü:**
  ![PR Direct Merge Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-30-29.png)

### 5. Görev Detayları ve Yorumlaşma (Task Details & Collaboration)
Oluşturulan görevlerin atamaları ve ekip içi yazışmalar görev detay paneli üzerinden gerçekleştirilir.
* **Görsel Tanımı:** Bir görev detayına tıklandığında açılan yan panelde görevin `melihemik` kullanıcısına atanması ve görev altındaki yorum geçmişinin görüntülenmesi.
* **Ekran Görüntüsü:**
  ![Task Details Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-30-14.png)

### 6. Telegram ve E-posta Bildirimleri (Telegram & Email Notifications)
Kullanıcılar kendilerine görev atandığında, bir görev tamamlandığında veya yorum yapıldığında anlık bildirim alırlar. Telegram bildirimleri için ayarlar sayfasından kullanıcıların kendi oluşturdukları bot tokenini ve Chat ID bilgilerini girmeleri yeterlidir.
* **Görsel Tanımı:** Kullanıcı ayarları sayfasındaki Telegram ve E-posta yapılandırma alanları ile bir kullanıcıya görev atandığında gönderilen şablon bildirim e-postası.
* **Ekran Görüntüleri:**
  ![Telegram Configuration Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-31-18.png)
  ![Email Notification Template Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-31-31.png)

### 7. İş Birlikçiler ve Rol Yetkilendirme (Collaborators & Role Access Control)
Depo bazlı işbirlikçilerin ve yetki düzeylerinin (Owner, Maintainer, Developer) yönetilmesini sağlar.
* **Görsel Tanımı:** "Repository Collaborators" yan paneli üzerinden depodaki işbirlikçilerin listelenmesi, rollerinin güncellenmesi ve yeni üye ekleme ekranı.
* **Ekran Görüntüsü:**
  ![Collaborators Screen](./screenshot/Ekran%20G%C3%B6r%C3%BCnt%C3%BCs%C3%BC%202026-06-19%2014-30-46.png)

## System Architecture

CodeTasker is composed of three services:
1. **Frontend:** A responsive Single Page Application built with React, TypeScript, TailwindCSS, and Vite.
2. **Backend:** A high-performance REST API built with Go, Fiber, and the official Google Go-GitHub client.
3. **Database:** MongoDB for storing user sessions, synced repository configurations, collaborators, tasks, and audit logs.

## Requirements

Ensure you have the following installed on your machine:
- **Go** (version 1.21 or later)
- **Node.js** (version 18 or later) & **npm**
- **MongoDB** (running locally or accessible via URI)

## Local Development Setup

### 1. Environment Configuration

Create a `.env` file in the root directory (and copy it to both `backend/` and `frontend/` directories as needed). Define the following variables:

```env
PORT=8080
MONGO_URI=mongodb://localhost:27017
DB_NAME=codetasker
GITHUB_CLIENT_ID=your_github_oauth_client_id
GITHUB_CLIENT_SECRET=your_github_oauth_client_secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/auth/github/callback
JWT_SECRET=your_jwt_signing_secret
WEBHOOK_SECRET=your_github_webhook_hmac_secret
TOKEN_ENCRYPT_KEY=your_aes_32byte_encryption_key
FRONTEND_URL=http://localhost:5173
```

### 2. Run the Backend

```bash
cd backend
go run cmd/server/main.go
```

### 3. Run the Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend will start at `http://localhost:5173/` and proxy API requests to `http://localhost:8080`.

## Quick Installation & Docker Setup

To quickly configure environment variables and launch the entire platform (MongoDB + Go API Backend + React Frontend) using Docker, run the interactive installation script at the project root:

```bash
./setup.sh
```

This script will:
1. Verify system prerequisites (`docker` and `docker compose`).
2. Prompt you step-by-step for configuration settings (Port, Database name, GitHub OAuth credentials, SMTP/email settings, etc.).
3. Auto-generate secure cryptographic keys if left blank (JWT secrets, AES-256 token encryption keys, webhook secrets).
4. Save the configuration to `.env` and offer to run the Docker Compose environment in the background automatically.

Alternatively, you can manually build and start the containers after creating your `.env` configuration:

```bash
docker compose up -d --build
```

## Contributing

Please review the [CONTRIBUTING.md](CONTRIBUTING.md) file for details on our code of conduct and the process for submitting pull requests.
