/**
 * @file 保持页面活跃 - 防止页面休眠导致API调用超时
 */
console.log('[keep_alive.js] 加载页面保活模块');

window.__wx_keep_alive = {
    wakeLock: null,
    activityTimer: null,
    visibilityTimer: null,

    // 初始化
    init: function () {
        console.log('[页面保活] 启动保活机制...');

        // 方法1: 使用 Wake Lock API 防止屏幕休眠（仅支持HTTPS或localhost）
        this.requestWakeLock();

        // 方法2: 定期模拟用户活动
        this.startActivitySimulation();

        // 方法3: 监听页面可见性变化，失去焦点时发出警告
        this.setupVisibilityMonitor();

        // 方法4: 定期执行轻量级DOM操作保持页面活跃
        this.startDOMActivity();
    },

    // 请求 Wake Lock（防止屏幕休眠）
    requestWakeLock: async function () {
        if (!('wakeLock' in navigator)) {
            console.log('[页面保活] ⚠️ 浏览器不支持 Wake Lock API');
            return;
        }

        try {
            this.wakeLock = await navigator.wakeLock.request('screen');
            console.log('[页面保活] ✅ Wake Lock 已激活');

            // 监听释放事件
            this.wakeLock.addEventListener('release', () => {
                console.log('[页面保活] ⚠️ Wake Lock 已释放，尝试重新获取...');
                // 延迟重新获取，避免频繁请求
                setTimeout(() => {
                    this.requestWakeLock();
                }, 1000);
            });
        } catch (err) {
            console.error('[页面保活] ❌ Wake Lock 请求失败:', err.message);
        }
    },

    // 模拟用户活动（定期触发一些无害的事件）
    startActivitySimulation: function () {
        var self = this;

        // 每30秒触发一次活动
        this.activityTimer = setInterval(function () {
            // 方法A: 更新页面标题（无害且不可见）
            var originalTitle = document.title;
            document.title = document.title; // 触发更新

            // 方法B: 触发一个自定义事件
            var event = new CustomEvent('wx_keep_alive_ping', {
                detail: { timestamp: Date.now() }
            });
            document.dispatchEvent(event);

            // 方法C: 读取一个DOM属性（触发渲染引擎）
            var _ = document.body.offsetHeight;

            // console.log('[页面保活] 💓 活动心跳 (30s)');
        }, 30000); // 30秒

        console.log('[页面保活] ✅ 活动模拟已启动 (30秒间隔)');
    },

    // 监听页面可见性
    setupVisibilityMonitor: function () {
        var self = this;

        document.addEventListener('visibilitychange', function () {
            if (document.hidden) {
                console.warn('[页面保活] ⚠️⚠️⚠️ 页面已隐藏！API调用可能失败！');
                console.warn('[页面保活] 请保持此标签页为活跃状态');

                // 尝试重新获取 Wake Lock
                if (self.wakeLock) {
                    self.wakeLock.release();
                    self.wakeLock = null;
                }
            } else {
                console.log('[页面保活] ✅ 页面已重新激活');

                // 页面重新可见时，重新请求 Wake Lock
                self.requestWakeLock();
            }
        });

        console.log('[页面保活] ✅ 可见性监控已启动');
    },

    // 定期执行轻量级DOM操作
    startDOMActivity: function () {
        var self = this;

        // 创建一个隐藏的div用于DOM操作
        var keepAliveDiv = document.createElement('div');
        keepAliveDiv.id = '__wx_keep_alive_marker';
        keepAliveDiv.style.display = 'none';
        keepAliveDiv.setAttribute('data-timestamp', Date.now());
        document.body.appendChild(keepAliveDiv);

        // 每10秒更新一次
        setInterval(function () {
            var marker = document.getElementById('__wx_keep_alive_marker');
            if (marker) {
                marker.setAttribute('data-timestamp', Date.now());
                // 触发重绘
                marker.offsetHeight;
            }
        }, 10000); // 10秒

        console.log('[页面保活] ✅ DOM活动已启动 (10秒间隔)');
    },

    // 停止保活（如果需要）
    stop: function () {
        console.log('[页面保活] 停止保活机制');

        if (this.wakeLock) {
            this.wakeLock.release();
            this.wakeLock = null;
        }

        if (this.activityTimer) {
            clearInterval(this.activityTimer);
            this.activityTimer = null;
        }

        var marker = document.getElementById('__wx_keep_alive_marker');
        if (marker) {
            marker.remove();
        }
    }
};

// 自动启动
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
        window.__wx_keep_alive.init();
    });
} else {
    window.__wx_keep_alive.init();
}

console.log('[keep_alive.js] 页面保活模块加载完成');
