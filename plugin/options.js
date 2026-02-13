document.addEventListener('DOMContentLoaded', () => {
	const portInput = document.getElementById("port");
	const browserPlaybackInput = document.getElementById("browserPlayback");
	const status = document.getElementById("status");

	chrome.storage.sync.get(["port", "browserPlayback"], (result) => {
		portInput.value = Number.isInteger(result.port) ? result.port : 54345;
		browserPlaybackInput.checked = typeof result.browserPlayback === "boolean" ? result.browserPlayback : false;
	});

	document.getElementById("save").addEventListener("click", () => {
		const port = parseInt(portInput.value, 10);
		const browserPlayback = browserPlaybackInput.checked;

		chrome.storage.sync.set({ port, browserPlayback }, () => {
			status.textContent = "options saved";
			setTimeout(() => { status.textContent = ""; }, 2000);
		});
	});
});
