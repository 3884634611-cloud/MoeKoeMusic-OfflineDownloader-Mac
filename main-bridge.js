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

    // === ODL_SAVE_DIR_PATCH: 系统文件夹选择框 ===
    // preload 在主世界挂了 electronAPI.showOpenDialog，隔离世界的 content.js
    // 通过本桥请求它，从而弹出原生「打开文件夹」面板（可选外接盘 / U 盘）。
    if (msg.path === '__odl_show_open_dialog') {
      if (!window.electronAPI || typeof window.electronAPI.showOpenDialog !== 'function') {
        reply({ __err: '当前客户端未暴露 electronAPI.showOpenDialog' });
        return;
      }
      var opts = msg.payload || {};
      Promise.resolve()
        .then(function () {
          return window.electronAPI.showOpenDialog({
            properties: opts.properties || ['openDirectory', 'createDirectory']
          });
        })
        .then(function (res) { reply({ res: res }); })
        .catch(function (e) { reply({ __err: String((e && e.message) || e) }); });
      return;
    }

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
