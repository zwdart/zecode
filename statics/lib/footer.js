/* =========================================================
 * zeco 通用组件：页脚 / 模态框 / Toast / 主题切换
 *
 * 用法（页面底部）：
 *     <script src="lib/footer.js"></script>
 *
 * 可选：在引入前覆盖站点信息
 *     <script>
 *         window.ZECO_SITE = { name: 'xxx', ... };
 *     </script>
 *     <script src="lib/footer.js"></script>
 *
 * 暴露 API：
 *     Zeco.showToast(msg)
 *     Zeco.toggleTheme()
 *     Zeco.openModal(html)
 *     Zeco.closeModal()
 *     Zeco.SITE
 * ========================================================= */
(function () {
    'use strict';

    /* ================= 站点配置 ================= */
    const SITE = Object.assign({
        name: 'zeco',
        about: `
            <p><strong>zeco</strong> 是一个极简的在线二维码工具，支持二维码的生成与识别。</p>
            <p>生成：输入任意文本或网址，一键生成二维码并下载 PNG。</p>
            <p>识别：上传二维码图片，自动解析出原始内容并支持复制。</p>
        `,
        links: [
            { name: '关于我们', url: 'https://zebra.dart.xin', desc: '主页' },
            { name: '一个小店', url: 'https://fone.taobao.com', desc: '淘宝小店' },
        ],
        beian: {
            // icp:    { text: '鄂ICP备2026xxxxx号-1', url: 'https://beian.miit.gov.cn/' },
            // police: { text: '浙公网安备330109xxxx号', url: 'https://beian.mps.gov.cn/' },
        },
        copyright: '© 2026 zeco · Zeco Code, Light Mode.',
    }, window.ZECO_SITE || {});

    /* ================= 主题切换 ================= */
    const THEME_KEY = 'zeco-theme';

    function applyTheme(theme) {
        document.documentElement.dataset.theme = theme;
    }

    function toggleTheme() {
        const cur = document.documentElement.dataset.theme || 'light';
        const next = cur === 'dark' ? 'light' : 'dark';
        applyTheme(next);
        try { localStorage.setItem(THEME_KEY, next); } catch (e) {}
    }

    // 系统主题变化：仅用户未手动设置时跟随
    (function watchSystemTheme() {
        if (!window.matchMedia) return;
        const mq = window.matchMedia('(prefers-color-scheme: dark)');
        const handler = (e) => {
            let saved = null;
            try { saved = localStorage.getItem(THEME_KEY); } catch (err) {}
            if (!saved) applyTheme(e.matches ? 'dark' : 'light');
        };
        if (mq.addEventListener) mq.addEventListener('change', handler);
        else if (mq.addListener) mq.addListener(handler);
    })();

    /* ================= 图标 ================= */
    const ICON_MOON = '<svg class="zeco-theme-icon zeco-theme-icon-moon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3a9 9 0 109 9 7 7 0 01-9-9z"/></svg>';
    const ICON_SUN  = '<svg class="zeco-theme-icon zeco-theme-icon-sun" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 7a5 5 0 100 10 5 5 0 000-10zm0-3a1 1 0 011 1v2a1 1 0 11-2 0V5a1 1 0 011-1zm0 14a1 1 0 011 1v2a1 1 0 11-2 0v-2a1 1 0 011-1zM4 12a1 1 0 011-1h2a1 1 0 110 2H5a1 1 0 01-1-1zm14 0a1 1 0 011-1h2a1 1 0 110 2h-2a1 1 0 01-1-1zM6.34 6.34a1 1 0 011.42 0l1.41 1.42a1 1 0 11-1.41 1.41L6.34 7.76a1 1 0 010-1.42zm9.9 9.9a1 1 0 011.41 0l1.42 1.41a1 1 0 11-1.42 1.42l-1.41-1.42a1 1 0 010-1.41zM6.34 17.66a1 1 0 010-1.41l1.42-1.42a1 1 0 111.41 1.42l-1.41 1.41a1 1 0 01-1.42 0zm9.9-9.9a1 1 0 010-1.41l1.41-1.42a1 1 0 111.42 1.42l-1.42 1.41a1 1 0 01-1.41 0z"/></svg>';

    /* ================= 注入 CSS ================= */
    const COMMON_CSS = `
        /* 主题切换按钮 */
        .zeco-theme-btn {
            position: fixed;
            top: 16px;
            right: var(--zeco-theme-btn-right, max(16px, calc(50% - 450px - 56px)));
            width: 40px;
            height: 40px;
            padding: 0;
            margin: 0;
            background: var(--glass-bg, rgba(255,255,255,0.95));
            backdrop-filter: blur(10px);
            color: var(--text-sub, #6b7280);
            border: 1px solid var(--border-color, #e5e7eb);
            border-radius: 10px;
            box-shadow: var(--shadow, 0 10px 25px -5px rgba(0,0,0,0.1));
            cursor: pointer;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            transition: all 0.2s;
            z-index: 80;
        }
        .zeco-theme-btn:hover {
            color: var(--primary, #4f46e5);
            border-color: var(--primary, #4f46e5);
            transform: translateY(-1px);
        }
        .zeco-theme-icon {
            width: 18px;
            height: 18px;
            fill: currentColor;
            display: block;
        }
        html[data-theme="light"] .zeco-theme-icon-sun  { display: none; }
        html[data-theme="dark"]  .zeco-theme-icon-moon { display: none; }

        /* 页脚 */
        .zeco-footer {
            width: 100%;
            max-width: 900px;
            margin-top: 30px;
            padding: 20px 10px 10px;
            text-align: center;
            color: var(--text-sub, #6b7280);
            font-size: 0.85rem;
            line-height: 1.9;
        }
        .zeco-footer-row {
            display: flex;
            flex-wrap: wrap;
            justify-content: center;
            align-items: center;
            gap: 6px 10px;
            margin-bottom: 4px;
        }
        .zeco-footer a {
            color: var(--text-sub, #6b7280);
            text-decoration: none;
            transition: color 0.2s;
            cursor: pointer;
        }
        .zeco-footer a:hover { color: var(--primary, #4f46e5); }
        .zeco-divider {
            color: var(--divider, #cbd5e1);
            user-select: none;
        }
        .zeco-footer-copy {
            color: var(--text-mute, #9ca3af);
            margin-top: 2px;
            font-size: 0.8rem;
        }
        .zeco-beian-item {
            display: inline-flex;
            align-items: center;
            gap: 4px;
        }
        .zeco-police-icon {
            width: 14px;
            height: 14px;
            display: inline-block;
            background: var(--police-icon, #94a3b8);
            border-radius: 2px;
            flex-shrink: 0;
        }

        /* 模态框 */
        .zeco-modal-mask {
            position: fixed;
            inset: 0;
            background: var(--modal-mask-bg, rgba(17,24,39,0.45));
            backdrop-filter: blur(4px);
            display: none;
            align-items: center;
            justify-content: center;
            padding: 20px;
            z-index: 90;
        }
        .zeco-modal-mask.show {
            display: flex;
            animation: zecoFadeIn 0.2s ease;
        }
        .zeco-modal {
            background: var(--surface-solid, #fff);
            color: var(--text-main, #1f2937);
            border-radius: var(--radius, 16px);
            box-shadow: 0 20px 40px -10px rgba(0,0,0,0.35);
            width: 100%;
            max-width: 520px;
            max-height: 80vh;
            overflow: auto;
            padding: 26px 24px 22px;
            position: relative;
            animation: zecoFadeIn 0.25s ease;
        }
        .zeco-modal h2 {
            font-size: 1.15rem;
            color: var(--primary, #4f46e5);
            margin-bottom: 14px;
            padding-right: 30px;
        }
        .zeco-modal p {
            color: var(--text-main, #1f2937);
            line-height: 1.75;
            margin-bottom: 10px;
            font-size: 0.95rem;
        }
        .zeco-modal-close {
            position: absolute;
            top: 12px;
            right: 12px;
            width: 32px;
            height: 32px;
            padding: 0;
            margin: 0;
            border-radius: 50%;
            background: var(--surface-muted, #f3f4f6);
            color: var(--text-sub, #6b7280);
            font-size: 20px;
            line-height: 1;
            cursor: pointer;
            border: none;
            transition: all 0.2s;
        }
        .zeco-modal-close:hover {
            background: var(--border-strong, #d1d5db);
            color: var(--text-main, #1f2937);
        }
        .zeco-link-list {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-top: 12px;
        }
        .zeco-link-item {
            display: block;
            padding: 12px 14px;
            border: 1px solid var(--border-color, #e5e7eb);
            border-radius: 10px;
            text-decoration: none;
            color: var(--text-main, #1f2937);
            transition: all 0.2s;
        }
        .zeco-link-item:hover {
            border-color: var(--primary, #4f46e5);
            background: var(--surface-hover, #eef2ff);
            color: var(--primary, #4f46e5);
        }
        .zeco-link-name {
            font-weight: 600;
            font-size: 0.95rem;
        }
        .zeco-link-desc {
            display: block;
            font-size: 0.8rem;
            color: var(--text-sub, #6b7280);
            margin-top: 3px;
        }

        /* Toast */
        .zeco-toast {
            position: fixed;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            background: var(--toast-bg, #1f2937);
            color: var(--toast-text, #fff);
            padding: 10px 20px;
            border-radius: 50px;
            font-size: 0.9rem;
            opacity: 0;
            transition: opacity 0.3s;
            pointer-events: none;
            z-index: 100;
            max-width: 90vw;
            text-align: center;
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        }
        .zeco-toast.show { opacity: 1; }

        @keyframes zecoFadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to   { opacity: 1; transform: translateY(0); }
        }
    `;

    /* ================= 挂载 ================= */
    function injectCSS() {
        if (document.getElementById('zeco-common-style')) return;
        const style = document.createElement('style');
        style.id = 'zeco-common-style';
        style.textContent = COMMON_CSS;
        document.head.appendChild(style);
    }

    function mountThemeBtn() {
        if (document.getElementById('zeco-theme-btn')) return;
        const btn = document.createElement('button');
        btn.id = 'zeco-theme-btn';
        btn.className = 'zeco-theme-btn';
        btn.setAttribute('aria-label', '切换主题');
        btn.setAttribute('title', '切换主题');
        btn.innerHTML = ICON_MOON + ICON_SUN;
        btn.addEventListener('click', toggleTheme);
        document.body.appendChild(btn);
    }

    function mountFooter() {
        if (document.getElementById('zeco-footer')) return;

        const footer = document.createElement('footer');
        footer.id = 'zeco-footer';
        footer.className = 'zeco-footer';
        footer.innerHTML = `
            <div class="zeco-footer-row">
                <a href="#" data-action="about">关于</a>
                <span class="zeco-divider">·</span>
                <a href="https://fone.taobao.com" target="_blank" rel="noopener">小店</a>
                <span class="zeco-divider">·</span>
                <a href="#" data-action="links">友链</a>
            </div>
            <div class="zeco-footer-row" id="zeco-footer-beian"></div>
            <div class="zeco-footer-row zeco-footer-copy" id="zeco-footer-copy"></div>
        `;
        footer.addEventListener('click', function (e) {
            const a = e.target.closest('[data-action]');
            if (!a) return;
            e.preventDefault();
            if (a.dataset.action === 'about') showAbout();
            else if (a.dataset.action === 'links') showLinks();
        });
        document.body.appendChild(footer);

        renderFooterContent();
    }

    function mountModal() {
        if (document.getElementById('zeco-modal-mask')) return;
        const mask = document.createElement('div');
        mask.id = 'zeco-modal-mask';
        mask.className = 'zeco-modal-mask';
        mask.innerHTML = `
            <div class="zeco-modal" role="dialog" aria-modal="true">
                <button class="zeco-modal-close" aria-label="关闭">&times;</button>
                <div id="zeco-modal-body"></div>
            </div>
        `;
        mask.addEventListener('click', function (e) {
            if (e.target === mask) closeModal();
        });
        mask.querySelector('.zeco-modal-close').addEventListener('click', closeModal);
        document.body.appendChild(mask);
    }

    function mountToast() {
        if (document.getElementById('zeco-toast')) return;
        const toast = document.createElement('div');
        toast.id = 'zeco-toast';
        toast.className = 'zeco-toast';
        document.body.appendChild(toast);
    }

    /* ================= 渲染页脚内容 ================= */
    function renderFooterContent() {
        const beianEl = document.getElementById('zeco-footer-beian');
        if (beianEl) {
            const items = [];
            if (SITE.beian.icp && SITE.beian.icp.text) {
                items.push('<a class="zeco-beian-item" href="' + SITE.beian.icp.url + '" target="_blank" rel="noopener">' + SITE.beian.icp.text + '</a>');
            }
            if (SITE.beian.police && SITE.beian.police.text) {
                items.push('<a class="zeco-beian-item" href="' + SITE.beian.police.url + '" target="_blank" rel="noopener"><span class="zeco-police-icon"></span>' + SITE.beian.police.text + '</a>');
            }
            beianEl.innerHTML = items.length
                ? items.join('<span class="zeco-divider">·</span>')
                : '';
        }
        const copyEl = document.getElementById('zeco-footer-copy');
        if (copyEl) copyEl.textContent = SITE.copyright;
    }

    /* ================= Toast ================= */
    let toastTimer = null;
    function showToast(msg) {
        const t = document.getElementById('zeco-toast');
        if (!t) return;
        t.innerText = msg;
        t.classList.add('show');
        clearTimeout(toastTimer);
        toastTimer = setTimeout(() => t.classList.remove('show'), 2000);
    }

    /* ================= 模态框 ================= */
    function openModal(html) {
        const body = document.getElementById('zeco-modal-body');
        const mask = document.getElementById('zeco-modal-mask');
        if (!body || !mask) return;
        body.innerHTML = html;
        mask.classList.add('show');
        document.body.style.overflow = 'hidden';
    }

    function closeModal() {
        const mask = document.getElementById('zeco-modal-mask');
        if (!mask) return;
        mask.classList.remove('show');
        document.body.style.overflow = '';
    }

    function escapeHtml(s) {
        return String(s).replace(/[&<>"']/g, function (c) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
        });
    }

    function showAbout() {
        openModal('<h2>关于 ' + escapeHtml(SITE.name) + '</h2>' + SITE.about);
    }

    function showLinks() {
        const list = (SITE.links && SITE.links.length)
            ? SITE.links.map(function (l) {
                return '<a class="zeco-link-item" href="' + escapeHtml(l.url) + '" target="_blank" rel="noopener">'
                    + '<span class="zeco-link-name">' + escapeHtml(l.name) + '</span>'
                    + (l.desc ? '<span class="zeco-link-desc">' + escapeHtml(l.desc) + '</span>' : '')
                    + '</a>';
            }).join('')
            : '<p>暂无友链，欢迎交换。</p>';
        openModal('<h2>友情链接</h2><div class="zeco-link-list">' + list + '</div>');
    }

    /* ================= 初始化 ================= */
    function init() {
        injectCSS();
        mountThemeBtn();
        mountFooter();
        mountModal();
        mountToast();

        document.addEventListener('keydown', function (e) {
            if (e.key === 'Escape') closeModal();
        });
    }

    /* ================= 暴露 API ================= */
    window.Zeco = {
        SITE: SITE,
        applyTheme: applyTheme,
        toggleTheme: toggleTheme,
        showToast: showToast,
        openModal: openModal,
        closeModal: closeModal,
    };
    // 兼容直接调用 showToast(...)
    window.showToast = showToast;

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();