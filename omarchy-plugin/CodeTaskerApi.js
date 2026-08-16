// CodeTaskerApi.js — Omarchy Plugin API & Storage Helper

var STORAGE_FILE = ".codetasker_omarchy_config.json";

function getStoragePath() {
    return (typeof localStorage !== "undefined") ? "local" : "config";
}

function getSetting(key, defaultValue) {
    if (typeof localStorage !== "undefined" && localStorage.getItem) {
        var val = localStorage.getItem("codetasker_" + key);
        return val !== null ? val : defaultValue;
    }
    return defaultValue;
}

function setSetting(key, value) {
    if (typeof localStorage !== "undefined" && localStorage.setItem) {
        localStorage.setItem("codetasker_" + key, value);
    }
}

function fetchNotifications(serverUrl, appToken, callback) {
    if (!serverUrl || !appToken) {
        callback(new Error("Missing Server URL or App Token"), null);
        return;
    }

    var baseUrl = serverUrl.replace(/\/+$/, "");
    var url = baseUrl + "/api/notifications";

    var xhr = new XMLHttpRequest();
    xhr.open("GET", url, true);
    xhr.setRequestHeader("X-App-Token", appToken);
    xhr.setRequestHeader("Accept", "application/json");

    xhr.onreadystatechange = function() {
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 200) {
                try {
                    var data = JSON.parse(xhr.responseText);
                    callback(null, data.notifications || []);
                } catch (e) {
                    callback(e, null);
                }
            } else if (xhr.status === 401 || xhr.status === 403) {
                callback(new Error("Invalid or Restricted App Token"), null);
            } else {
                callback(new Error("HTTP Error " + xhr.status), null);
            }
        }
    };
    xhr.send();
}

function fetchUnreadCount(serverUrl, appToken, callback) {
    if (!serverUrl || !appToken) {
        callback(null, 0);
        return;
    }

    var baseUrl = serverUrl.replace(/\/+$/, "");
    var url = baseUrl + "/api/notifications/unread-count";

    var xhr = new XMLHttpRequest();
    xhr.open("GET", url, true);
    xhr.setRequestHeader("X-App-Token", appToken);
    xhr.setRequestHeader("Accept", "application/json");

    xhr.onreadystatechange = function() {
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 200) {
                try {
                    var data = JSON.parse(xhr.responseText);
                    callback(null, data.count || 0);
                } catch (e) {
                    callback(e, 0);
                }
            } else {
                callback(new Error("HTTP " + xhr.status), 0);
            }
        }
    };
    xhr.send();
}

function markAllRead(serverUrl, appToken, callback) {
    if (!serverUrl || !appToken) return;

    var baseUrl = serverUrl.replace(/\/+$/, "");
    var url = baseUrl + "/api/notifications/read-all";

    var xhr = new XMLHttpRequest();
    xhr.open("PATCH", url, true);
    xhr.setRequestHeader("X-App-Token", appToken);
    xhr.setRequestHeader("Accept", "application/json");

    xhr.onreadystatechange = function() {
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 200) {
                callback(null);
            } else {
                callback(new Error("HTTP " + xhr.status));
            }
        }
    };
    xhr.send();
}

function markRead(serverUrl, appToken, id, callback) {
    if (!serverUrl || !appToken || !id) return;

    var baseUrl = serverUrl.replace(/\/+$/, "");
    var url = baseUrl + "/api/notifications/" + id + "/read";

    var xhr = new XMLHttpRequest();
    xhr.open("PATCH", url, true);
    xhr.setRequestHeader("X-App-Token", appToken);
    xhr.setRequestHeader("Accept", "application/json");

    xhr.onreadystatechange = function() {
        if (xhr.readyState === XMLHttpRequest.DONE) {
            if (xhr.status === 200) {
                callback(null);
            } else {
                callback(new Error("HTTP " + xhr.status));
            }
        }
    };
    xhr.send();
}
