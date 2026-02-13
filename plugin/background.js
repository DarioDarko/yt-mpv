const ytRegex = /^(https?:\/\/)?(www\.)?youtube\.com\/watch/;
let lastSentUrl = null;

chrome.tabs.onUpdated.addListener(async (tabId, changeInfo, tab) => {
	try {
		if (changeInfo.url || changeInfo.status === "complete") {
			const activeTabs = await chrome.tabs.query({ active: true, windowId: tab.windowId });
			const isActiveTab = activeTabs.some(activeTab => activeTab.id === tabId);
			
			if (!isActiveTab) {
				return;
			}

			const url = changeInfo.url || tab.url;

			if (url === lastSentUrl) {
				return;
			}

			lastSentUrl = url;

			if (!ytRegex.test(url)) {
				return;
			}

			const response = await fetch("http://localhost:54345/play", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ url })
			});

			if (!response.ok) {
				console.error("Failed to communicate with yt-mpv server");
			}
		}
	} catch (err) {
		console.error("Error connecting to yt-mpv server", err);
	}
});

chrome.commands.onCommand.addListener(async (command) => {
	try {
		const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

		if (!tab) return;
		if (!ytRegex.test(tab.url)) return;

		chrome.tabs.sendMessage(tab.id, { action: command }, (response) => {
			if (chrome.runtime.lastError) {
				console.warn("Content script might not be injected yet:", chrome.runtime.lastError.message);
			}
		});
	} catch (err) {
		console.error("Error when handling media command:", err);
	}
});
