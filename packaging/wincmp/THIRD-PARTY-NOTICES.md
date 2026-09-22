# Third-Party Software Notices and Information

This file contains notices and license information for third-party software components and dependencies utilized, managed, or integrated by WinCMP.

---

## Trademark & Non-Endorsement Disclaimer / 商標與免責聲明

WinCMP is an independent open-source project. All product names, logos, brands, and trademarks displayed or referred to within the software, website, and documentation are the property of their respective trademark holders.

These trademark holders are not affiliated with WinCMP, and they do not sponsor, endorse, or promote WinCMP in any manner.

WinCMP 是一個獨立的開源開發環境管理工具。軟體、網站與文檔中提及之產品名稱、標誌、品牌與商標（包括但不限於 Caddy, MariaDB, Redis, PHP, Node.js, Composer, HeidiSQL, Mailpit, Bun 等）均屬其各自商標權利人所有。各商標持有人未與 WinCMP 存在附屬關係，亦未贊助、認可或背書 WinCMP。

---

## WinCMP Architecture & Distribution Boundary / 分發與合規邊界說明

1. **WinCMP Core**:
   WinCMP core application and configuration files are distributed under the **MIT License**.
   WinCMP 核心主程式與設定檔案採用 **MIT 授權** 分發。

2. **Downloader / Orchestrator Model (Default)**:
   The official WinCMP release binary packages (`wincmp-v*.zip`, `WinCMP_v*.exe`) **do not bundle** external third-party service binaries. External runtimes (such as MariaDB, Redis, PHP, Caddy, Node.js, etc.) are downloaded directly from their respective official release servers by the user upon request via WinCMP's built-in dependency manager. WinCMP acts solely as an orchestrator and process supervisor, and therefore does not constitute binary redistribution of these services.
   WinCMP 官方發布包 **未捆綁分發** 外部服務之二進位編譯檔案。各項執行環境係由使用者透過內建依賴下載器自官方伺服器直接取得。WinCMP 僅作為調度器與行程管理器，不構成對外部服務之二進位重新分發。

3. **Source Code Obligations for Copyleft Components**:
   For transparency and community adherence, official source code repositories and license references for all managed components (including GPL-licensed components like MariaDB and HeidiSQL) are explicitly provided below.

---

## 1. Managed Runtimes & External Services / 受控執行環境與外部服務

| Component | License (SPDX) | Official Website | Source Code Repository |
|---|---|---|---|
| **Caddy** | Apache-2.0 | https://caddyserver.com | https://github.com/caddyserver/caddy |
| **MariaDB** | GPL-2.0-only / LGPL-2.1-only | https://mariadb.org | https://github.com/MariaDB/server |
| **Redis (Windows Port)** | BSD-3-Clause | https://redis.io | https://github.com/tporadowski/redis |
| **PHP** | PHP-3.01 | https://www.php.net | https://github.com/php/php-src |
| **PHP Redis Extension** | PHP-3.01 | https://pecl.php.net/package/redis | https://github.com/phpredis/phpredis |
| **Mailpit** | MIT | https://mailpit.axllent.org | https://github.com/axllent/mailpit |
| **Node.js** | MIT (and subcomponent licenses) | https://nodejs.org | https://github.com/nodejs/node |
| **Composer** | MIT | https://getcomposer.org | https://github.com/composer/composer |
| **HeidiSQL** | GPL-3.0-or-later | https://www.heidisql.com | https://github.com/HeidiSQL/HeidiSQL |
| **Bun (Optional)** | MIT | https://bun.sh | https://github.com/oven-sh/bun |
| **CA Certificates (cacert.pem)** | MPL-2.0 | https://curl.se/ca/ | https://github.com/curl/curl |

---

## 2. Go Backend Dependencies / 後端相依套件

WinCMP includes or links with the following Go packages under permissive open-source licenses:

- **Wails v2** (`github.com/wailsapp/wails/v2`) - MIT License  
  Copyright (c) Lea Anthony and Wails contributors
- **golang.org/x/sys** (`golang.org/x/sys`) - BSD-3-Clause License  
  Copyright (c) The Go Authors
- **go-ole** (`github.com/go-ole/go-ole`) - MIT License  
  Copyright (c) Yasuhiro Matsumoto
- **lumberjack** (`gopkg.in/natefinch/lumberjack.v2`) - MIT License  
  Copyright (c) Nate Finch
- **pty** (`github.com/creack/pty`) - MIT License  
  Copyright (c) Keith Rarick

---

## 3. Frontend Dependencies / 前端相依套件

The WinCMP frontend interface includes the following npm libraries:

- **React / React-DOM** (`react`, `react-dom`) - MIT License  
  Copyright (c) Meta Platforms, Inc. and affiliates.
- **Lucide React** (`lucide-react`) - ISC License  
  Copyright (c) Lucide Contributors
- **Zustand** (`zustand`) - MIT License  
  Copyright (c) Paul Henschel and contributors
- **xterm.js** (`xterm`, `xterm-addon-fit`, `xterm-addon-web-links`) - MIT License  
  Copyright (c) Microsoft Corporation
- **Tailwind CSS** (`tailwindcss`) - MIT License  
  Copyright (c) Tailwind Labs, Inc.
- **Vite** (`vite`) - MIT License  
  Copyright (c) Yuxi (Evan) You and Vite contributors

---

## 4. Key License Texts / 核心授權條款摘錄

### MIT License
```text
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Apache License 2.0
```text
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### BSD 3-Clause License
```text
Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software without
   specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
```

### GNU General Public License v2 (GPL-2.0)
MariaDB server is licensed under the GPL-2.0-only.
Official full license text: https://www.gnu.org/licenses/old-licenses/gpl-2.0.txt
Official corresponding source: https://github.com/MariaDB/server

### GNU General Public License v3 (GPL-3.0)
HeidiSQL is licensed under the GPL-3.0-or-later.
Official full license text: https://www.gnu.org/licenses/gpl-3.0.txt
Official corresponding source: https://github.com/HeidiSQL/HeidiSQL

### PHP License v3.01
PHP and PECL extensions are licensed under the PHP License v3.01.
Official full license text: https://www.php.net/license/3_01.txt
Official corresponding source: https://github.com/php/php-src
