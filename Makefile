samlare: *.go
	go build .

release: samlare
	@echo "current version: "
	@cat VERSION
	@read -p "Version: " version; \
	echo $$version > VERSION; \
	git commit -a -m "release version $$version"; \
	git tag -a v$$version -m "release $$version"; \
	git push --tags origin master

.PHONY: release
