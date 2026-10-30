/*
 * 列表每页显示条数
 *
 * 可选条数统一在这里维护，三个列表页（客户端、隧道、域名）共用；
 * 用户在分页栏手动切换后写入 localStorage，刷新或重新登录后仍然生效。
 */
(function () {
    var STORAGE_KEY = 'nps_page_size';
    var PAGE_LIST = [20, 50, 100, 200];
    var DEFAULT_PAGE_SIZE = PAGE_LIST[0];

    function normalize(size) {
        size = parseInt(size, 10);
        return PAGE_LIST.indexOf(size) === -1 ? DEFAULT_PAGE_SIZE : size;
    }

    window.npsPageSize = {
        list: PAGE_LIST,
        // 已保存的选择优先，其次使用默认值
        get: function () {
            var saved = null;
            try {
                saved = window.localStorage.getItem(STORAGE_KEY);
            } catch (e) {
                // 隐私模式等场景下 localStorage 不可用，忽略即可
            }
            return normalize(saved);
        },
        save: function (size) {
            try {
                window.localStorage.setItem(STORAGE_KEY, normalize(size));
            } catch (e) {
                // 同上，保存失败不影响本次浏览
            }
        }
    };
})();
