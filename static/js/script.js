// 假设用户信息来自某个 API 或静态数据
const userInfo = {
    username: "用户名称", // 这里可以替换为实际的用户数据
    email: "user@example.com" // 这里可以替换为实际的用户数据
};

// 更新用户信息显示
document.getElementById('username').querySelector('span').textContent = userInfo.username;
document.getElementById('email').querySelector('span').textContent = userInfo.email;

// WebSocket 连接
var ws = new WebSocket('ws://' + window.location.host + '/jb-server-page?reloadMode=RELOAD_ON_SAVE&referer=' + encodeURIComponent(window.location.pathname));

ws.onmessage = function (msg) {
    if (msg.data === 'reload') {
        window.location.reload();
    }

    if (msg.data.startsWith('update-css ')) {
        var messageId = msg.data.substring(11);
        var links = document.getElementsByTagName('link');

        for (var i = 0; i < links.length; i++) {
            var link = links[i];
            if (link.rel !== 'stylesheet') continue;

            var clonedLink = link.cloneNode(true);
            var newHref = link.href.replace(/(&|\\?)jbUpdateLinksId=\\d+/, "$1jbUpdateLinksId=" + messageId);

            if (newHref !== link.href) {
                clonedLink.href = newHref;
            } else {
                var indexOfQuest = newHref.indexOf('?');
                if (indexOfQuest >= 0) {
                    clonedLink.href = newHref.substring(0, indexOfQuest + 1) + 'jbUpdateLinksId=' + messageId + '&' + newHref.substring(indexOfQuest + 1);
                } else {
                    clonedLink.href += '?jbUpdateLinksId=' + messageId;
                }
            }

            link.parentNode.replaceChild(clonedLink, link);
        }
    }
};

// 文件上传逻辑
document.getElementById('uploadButton').addEventListener('click', function() {
    const fileInput = document.getElementById('fileInput');
    const fileList = document.getElementById('fileList');

    // 获取选中的文件
    const files = fileInput.files;

    // 遍历每个文件并添加到文件列表
    for (let i = 0; i < files.length; i++) {
        const listItem = document.createElement('li');
        listItem.textContent = files[i].name;
        fileList.appendChild(listItem);
    }

    // 清空文件输入框
    fileInput.value = '';
});
