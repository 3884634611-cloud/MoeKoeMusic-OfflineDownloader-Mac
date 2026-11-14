// 主世界桥接：隔离世界的 content.js 访问不到 preload 暴露在主世界的
// window.kugouapi，由本脚本（被隔离世界以 <script> 标签注入主世界执行）
// 代收 postMessage，转发给 kugouapi.request，再把结果 postMessage 回去。
(function () {
  'use strict';

  if (window.__odlBridgeInjected) return;
  window.__odlBridgeInjected = true;

  window.addEventListener('message', function (ev) {
    var msg = ev.data;
    if (!msg || msg.__odl !== 'bridge-req') return;

    var reply = function (payload) {
      window.postMessage({ __odl: 'bridge-res', id: msg.id, payload: payload }, '*');
    };

    if (!window.kugouapi || typeof window.kugouapi.request !== 'function') {
      reply({ __err: '当前客户端未注入 kugouapi，请升级 MoeKoeMusic 到支持内置 API 的版本' });
      return;
    }

    Promise.resolve()
      .then(function () { return window.kugouapi.request(msg.path, msg.payload || {}); })
      .then(function (res) { reply({ res: res }); })
      .catch(function (e) { reply({ __err: String((e && e.message) || e) }); });
  });
})();
