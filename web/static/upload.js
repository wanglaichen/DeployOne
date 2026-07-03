(function () {
  var form = document.querySelector(".upload-form");
  var modal = document.getElementById("upload-modal");
  if (!form || !modal) {
    return;
  }

  var category = form.dataset.category || "android";
  var fileInput = document.getElementById("upload-file");
  var openButtons = document.querySelectorAll("#open-upload-modal, [data-open-upload]");
  var closeButtons = modal.querySelectorAll("[data-close-modal]");
  var modalClose = modal.querySelector(".modal-close");
  var dropzone = document.getElementById("modal-dropzone");
  var idleState = document.getElementById("modal-state-idle");
  var uploadingState = document.getElementById("modal-state-uploading");
  var cancelButton = document.getElementById("upload-cancel");
  var statusEl = document.getElementById("upload-status");
  var percentEl = document.getElementById("upload-percent");
  var barEl = document.getElementById("upload-progress-bar");
  var bytesEl = document.getElementById("upload-bytes");
  var speedEl = document.getElementById("upload-speed");
  var uploadNameEl = document.getElementById("modal-upload-name");
  var uploadSizeEl = document.getElementById("modal-upload-size");
  var uploadIconEl = document.getElementById("modal-upload-icon");
  var dropzoneIconEl = document.getElementById("modal-dropzone-icon");

  var uploading = false;
  var activeXHR = null;
  var lastLoaded = 0;
  var lastTime = 0;

  var categoryIcons = {
    android: "🤖",
    ios: "🍎",
    file: "📁"
  };

  var messages = {
    android: {
      empty: "请选择 APK 文件",
      invalid: "只允许上传 .apk 文件"
    },
    ios: {
      empty: "请选择 IPA 文件",
      invalid: "只允许上传 .ipa 文件"
    },
    file: {
      empty: "请选择要上传的文件",
      invalid: "请上传有效的文件"
    }
  };

  function currentMessages() {
    return messages[category] || messages.android;
  }

  function categoryIcon() {
    return categoryIcons[category] || "📦";
  }

  function formatBytes(size) {
    if (!size || size < 0) {
      return "0 B";
    }
    var units = ["B", "KB", "MB", "GB"];
    var value = size;
    var unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
      value /= 1024;
      unitIndex += 1;
    }
    if (unitIndex === 0) {
      return Math.round(value) + " " + units[unitIndex];
    }
    return value.toFixed(1) + " " + units[unitIndex];
  }

  function formatSpeed(bytesPerSecond) {
    if (!bytesPerSecond || bytesPerSecond <= 0) {
      return "";
    }
    return formatBytes(bytesPerSecond) + "/s";
  }

  function isAllowedFile(name) {
    if (!name) {
      return false;
    }
    if (category === "android") {
      return /\.apk$/i.test(name);
    }
    if (category === "ios") {
      return /\.ipa$/i.test(name);
    }
    return /\.[^./\\]+$/i.test(name);
  }

  function setModalOpen(open) {
    modal.classList.toggle("hidden", !open);
    modal.setAttribute("aria-hidden", open ? "false" : "true");
    document.body.classList.toggle("modal-open", open);
  }

  function showIdleState() {
    idleState.classList.remove("hidden");
    uploadingState.classList.add("hidden");
    statusEl.classList.add("hidden");
    statusEl.textContent = "";
    barEl.style.width = "0%";
    barEl.classList.remove("is-indeterminate", "is-error", "is-complete");
    percentEl.textContent = "0%";
    bytesEl.textContent = "0 B / 0 B";
    speedEl.textContent = "";
    lastLoaded = 0;
    lastTime = 0;
  }

  function showUploadingState(file) {
    idleState.classList.add("hidden");
    uploadingState.classList.remove("hidden");
    uploadNameEl.textContent = file.name;
    uploadSizeEl.textContent = formatBytes(file.size);
    uploadIconEl.textContent = categoryIcon();
    dropzoneIconEl.textContent = categoryIcon();
    modalClose.disabled = true;
    cancelButton.disabled = false;
  }

  function setUploading(active) {
    uploading = active;
    fileInput.disabled = active;
    form.querySelectorAll(".modal-meta input").forEach(function (input) {
      input.disabled = active;
    });
    openButtons.forEach(function (button) {
      button.disabled = active;
    });
    if (!active) {
      modalClose.disabled = false;
    }
  }

  function updateProgress(loaded, total) {
    if (total > 0) {
      var percent = Math.min(100, Math.round((loaded / total) * 100));
      barEl.classList.remove("is-indeterminate");
      barEl.style.width = percent + "%";
      percentEl.textContent = percent + "%";
      bytesEl.textContent = formatBytes(loaded) + " / " + formatBytes(total);

      var now = Date.now();
      if (lastTime > 0) {
        var deltaBytes = loaded - lastLoaded;
        var deltaTime = (now - lastTime) / 1000;
        if (deltaTime > 0 && deltaBytes >= 0) {
          speedEl.textContent = formatSpeed(deltaBytes / deltaTime);
        }
      }
      lastLoaded = loaded;
      lastTime = now;
      return;
    }

    barEl.classList.add("is-indeterminate");
    barEl.style.width = "100%";
    percentEl.textContent = "上传中";
    bytesEl.textContent = formatBytes(loaded) + " / 未知大小";
    speedEl.textContent = "";
  }

  function showStatus(message, isError) {
    statusEl.textContent = message;
    statusEl.classList.remove("hidden");
    statusEl.classList.toggle("is-error", !!isError);
  }

  function resetModal() {
    if (activeXHR) {
      activeXHR.abort();
      activeXHR = null;
    }
    setUploading(false);
    showIdleState();
    fileInput.value = "";
  }

  function openModal() {
    if (uploading) {
      return;
    }
    showIdleState();
    setModalOpen(true);
  }

  function closeModal(force) {
    if (uploading && !force) {
      return;
    }
    resetModal();
    setModalOpen(false);
  }

  function startUpload(file) {
    var msg = currentMessages();
    if (!file) {
      window.alert(msg.empty);
      return;
    }
    if (!isAllowedFile(file.name)) {
      window.alert(msg.invalid);
      fileInput.value = "";
      return;
    }

    var formData = new FormData(form);
    var xhr = new XMLHttpRequest();
    activeXHR = xhr;

    setUploading(true);
    showUploadingState(file);
    updateProgress(0, file.size);

    xhr.open("POST", form.action || "/upload", true);
    xhr.upload.addEventListener("progress", function (event) {
      updateProgress(event.loaded, event.lengthComputable ? event.total : file.size);
    });

    xhr.addEventListener("load", function () {
      activeXHR = null;
      if (xhr.status >= 200 && xhr.status < 400) {
        barEl.classList.add("is-complete");
        barEl.style.width = "100%";
        percentEl.textContent = "100%";
        showStatus("上传成功，正在刷新页面…", false);
        window.setTimeout(function () {
          window.location.href = xhr.responseURL || "/?tab=" + category;
        }, 450);
        return;
      }

      barEl.classList.add("is-error");
      showStatus("上传失败，请重试", true);
      setUploading(false);
      cancelButton.disabled = false;
    });

    xhr.addEventListener("error", function () {
      activeXHR = null;
      barEl.classList.add("is-error");
      showStatus("网络错误，上传失败", true);
      setUploading(false);
      cancelButton.disabled = false;
    });

    xhr.addEventListener("abort", function () {
      activeXHR = null;
      resetModal();
    });

    xhr.send(formData);
  }

  openButtons.forEach(function (button) {
    button.addEventListener("click", openModal);
  });

  closeButtons.forEach(function (button) {
    button.addEventListener("click", function () {
      closeModal(false);
    });
  });

  cancelButton.addEventListener("click", function () {
    if (activeXHR) {
      activeXHR.abort();
    } else {
      resetModal();
    }
  });

  fileInput.addEventListener("change", function () {
    var file = fileInput.files && fileInput.files[0];
    if (file) {
      startUpload(file);
    }
  });

  ["dragenter", "dragover"].forEach(function (eventName) {
    dropzone.addEventListener(eventName, function (event) {
      event.preventDefault();
      dropzone.classList.add("is-dragover");
    });
  });

  ["dragleave", "drop"].forEach(function (eventName) {
    dropzone.addEventListener(eventName, function (event) {
      event.preventDefault();
      dropzone.classList.remove("is-dragover");
    });
  });

  dropzone.addEventListener("drop", function (event) {
    if (uploading) {
      return;
    }
    var files = event.dataTransfer && event.dataTransfer.files;
    if (!files || !files.length) {
      return;
    }
    fileInput.files = files;
    startUpload(files[0]);
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && !modal.classList.contains("hidden")) {
      closeModal(false);
    }
  });

  dropzoneIconEl.textContent = categoryIcon();
})();
