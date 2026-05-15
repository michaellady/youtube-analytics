.PHONY: hooks-install schedule-install schedule-uninstall schedule-test \
        launch-watch-install launch-watch-uninstall launch-watch-test \
        test vet check

REPO_DIR := $(shell pwd)
LAUNCH_AGENT_DIR := $(HOME)/Library/LaunchAgents

LAUNCH_AGENT_LABEL := com.mikelady.yt-weekly-review
LAUNCH_AGENT_PLIST := $(LAUNCH_AGENT_DIR)/$(LAUNCH_AGENT_LABEL).plist

LAUNCH_WATCH_LABEL := com.mikelady.yt-launch-watch
LAUNCH_WATCH_PLIST := $(LAUNCH_AGENT_DIR)/$(LAUNCH_WATCH_LABEL).plist

# Activate the tracked git hooks (.githooks/) for this clone.
# Run once after cloning. Idempotent.
hooks-install:
	git config --local core.hooksPath .githooks
	@echo "Pre-commit hook activated. To bypass: git commit --no-verify"

# Install the Sunday 9am weekly-review LaunchAgent. Materializes the
# template at scripts/com.mikelady.yt-weekly-review.plist with absolute
# paths into ~/Library/LaunchAgents/ and loads it with launchctl.
schedule-install:
	@mkdir -p $(LAUNCH_AGENT_DIR)
	@mkdir -p $(HOME)/Library/Logs/yt-weekly-review
	@sed -e 's|__REPO_DIR__|$(REPO_DIR)|g' -e 's|__HOME__|$(HOME)|g' \
		scripts/$(LAUNCH_AGENT_LABEL).plist > $(LAUNCH_AGENT_PLIST)
	@launchctl unload $(LAUNCH_AGENT_PLIST) 2>/dev/null || true
	@launchctl load $(LAUNCH_AGENT_PLIST)
	@echo "Installed $(LAUNCH_AGENT_LABEL) — fires Sundays at 9 AM local."
	@echo "Logs: ~/Library/Logs/yt-weekly-review/"
	@echo "Verify: launchctl list | grep yt-weekly-review"

# Remove the LaunchAgent. Idempotent.
schedule-uninstall:
	@launchctl unload $(LAUNCH_AGENT_PLIST) 2>/dev/null || true
	@rm -f $(LAUNCH_AGENT_PLIST)
	@echo "Uninstalled $(LAUNCH_AGENT_LABEL)."

# Run the weekly-review script immediately (does NOT wait for Sunday).
# Useful for testing the install end-to-end.
schedule-test:
	bash scripts/weekly-review.sh

# Daily launch-window monitor for a single video. Fires once daily
# at 9 AM local. Designed for the first ~1-2 weeks after a drop.
# Required: VIDEO=<video-id>
launch-watch-install:
	@if [ -z "$(VIDEO)" ]; then echo "Usage: make launch-watch-install VIDEO=<video-id>"; exit 2; fi
	@mkdir -p $(LAUNCH_AGENT_DIR)
	@mkdir -p $(HOME)/Library/Logs/yt-launch-watch
	@sed -e 's|__REPO_DIR__|$(REPO_DIR)|g' -e 's|__HOME__|$(HOME)|g' -e 's|__VIDEO_ID__|$(VIDEO)|g' \
		scripts/$(LAUNCH_WATCH_LABEL).plist > $(LAUNCH_WATCH_PLIST)
	@launchctl unload $(LAUNCH_WATCH_PLIST) 2>/dev/null || true
	@launchctl load $(LAUNCH_WATCH_PLIST)
	@echo "Installed $(LAUNCH_WATCH_LABEL) for video $(VIDEO) — fires daily at 9 AM local."
	@echo "Logs: ~/Library/Logs/yt-launch-watch/$(VIDEO)/"
	@echo "Run \`make launch-watch-uninstall\` after the launch window."

launch-watch-uninstall:
	@launchctl unload $(LAUNCH_WATCH_PLIST) 2>/dev/null || true
	@rm -f $(LAUNCH_WATCH_PLIST)
	@echo "Uninstalled $(LAUNCH_WATCH_LABEL)."

# Fire the launch-watch script immediately for VIDEO=<id>.
launch-watch-test:
	@if [ -z "$(VIDEO)" ]; then echo "Usage: make launch-watch-test VIDEO=<video-id>"; exit 2; fi
	bash scripts/launch-watch.sh $(VIDEO)

test:
	go test ./...

vet:
	go vet ./...

check: vet test
