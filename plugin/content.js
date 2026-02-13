const video = document.querySelector("video");
const playButton = document.querySelector(".ytp-play-button");
const seekBar = document.querySelector(".ytp-progress-bar");

if (video) {
	chrome.storage.sync.get(["port", "browserPlayback"], (result) => {
		const port = Number.isInteger(result.port) ? result.port : 54345;
		const browserPlayback = typeof result.browserPlayback === "boolean" ? result.browserPlayback : false;

		function sendPlayRequest() {
			fetch("http://localhost:54345/play-pause", {
				method: "POST",
				headers: { "Content-Type": "application/json" }
			}).then(response => {
				if (!response.ok) {
					console.error("Failed to send play-pause to yt-mpv server");
				}
			})
		}

		function sendSeekRequest(seconds) {
			fetch("http://localhost:54345/seek", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ seconds: seconds })
			}).then(response => {
				if (!response.ok) {
					console.error("Failed to send seek to yt-mpv server");
				}
			})
		}

		if (!browserPlayback) {
			const observer = new MutationObserver(() => {
				video.pause();
			});

			observer.observe(document.body, {
				childList: true,
				subtree: true
			});
		} else {
			const player = document.getElementById("movie_player");

			function syncYouTubeToMpvTime() {
				fetch("http://localhost:54345/time", { method: "POST" })
				.then(res => res.json())
				.then(data => {
					if (!data.time) {
						return;
					}

					const mpvTime = Math.round(data.time);
					video.currentTime = mpvTime
				})
			}

			setInterval(syncYouTubeToMpvTime, 5000)
		}

		video.addEventListener("click", (event) => {
			// event.stopImmediatePropagation();
			// event.preventDefault();

			sendPlayRequest();
		});

		playButton.addEventListener("click", (event) => {
			// event.stopImmediatePropagation();
			// event.preventDefault();

			sendPlayRequest();
		});

		seekBar.addEventListener("click", (event) => {
			const rect = seekBar.getBoundingClientRect();
			const clickX = event.clientX - rect.left;
			const width = rect.width;
			const percent = clickX / width;
			const seconds = percent * video.duration;
			const roundedSeconds = Math.round(seconds);

			sendSeekRequest(roundedSeconds);
		});
	});
}
