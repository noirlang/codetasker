# CodeTasker Notifications — Omarchy Quattro Plugin

A custom top-bar widget and popover notification panel for **Omarchy** (Linux Shell), styled in CodeTasker's dark theme.

![Omarchy Plugin](https://img.shields.io/badge/Omarchy-Quattro%20Plugin-10b981?style=flat-square)

---

## 🌟 Features

* **Top Bar Icon (`</>`)**:
  * **White (`#ffffff`)**: No unread notifications.
  * **Red (`#ef4444`)**: Unread notifications present (with numeric counter badge).
* **CodeTasker Dark Panel**:
  * Sleek popover window displaying live notifications.
  * Mark individual notifications as read or clear all.
* **App Token Authentication**:
  * Uses CodeTasker's scoped App Tokens (`notifications:read`).
  * **First-Run Prompt**: Prompts for Server URL and App Token on first click, saves it permanently, and never asks again.
  * Settings gear (⚙️) available in panel header to update token or server URL at any time.

---

## 🚀 Installation & Local Setup

### 1. Copy Plugin to Omarchy Config Directory

To install the plugin on your local system, copy the `omarchy-plugin` directory to your user's Omarchy plugins path:

```bash
mkdir -p ~/.config/omarchy/plugins/codetasker.notifications
cp -r omarchy-plugin/* ~/.config/omarchy/plugins/codetasker.notifications/
```

### 2. Rescan & Enable Plugin in Omarchy

Rescan plugins in Omarchy:

```bash
omarchy-shell shell rescanPlugins
```

Or validate the plugin manifest:

```bash
omarchy plugin validate ~/.config/omarchy/plugins/codetasker.notifications
```

---

## 🔑 Generating Your App Token

1. Open **CodeTasker** in your browser (`http://localhost:8080` or your deployment URL).
2. Go to **Dashboard → Settings → App Tokens (Notification API Keys)**.
3. Type a token name (e.g., `Omarchy Desktop`) and click **Generate Token**.
4. Copy the raw token (`ct_app_...`).
5. Click the `</>` icon in your Omarchy top bar, paste the token into the setup prompt, and click **Save Token & Connect**.

---

## 📂 File Sitemap

* `manifest.json` — Omarchy plugin metadata & entry points (`barWidget` & `panel`).
* `BarWidget.qml` — Top bar widget item with `</>` icon and unread count badge.
* `Panel.qml` — Popover panel with notification list and token configuration prompt.
* `CodeTaskerApi.js` — JavaScript HTTP client communicating with `/api/notifications` using `X-App-Token`.
