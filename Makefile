# XDProbe Makefile
# It handles downloading assets and building the project.

SRCS      := $(shell find . -maxdepth 1 -type f -name "*.go")
ASSETS    := assets/globe.gl assets/h3.js assets/alpine.js assets/ne_110m_admin_0_countries.json assets/Geist-Variable.woff2 assets/GeistMono-Variable.woff2 assets/favicon.svg assets/index.css
GREEN     := [\033[32m%10s\033[0m]

.PHONY: clean cleanall

xdprobe: $(SRCS) kernel/program_bpfel.go $(ASSETS)
	@go build -buildvcs=false  -o $@ . && printf "$(GREEN) \033[1m%s\033[0m\n" "BUILT" "$@"

kernel/program_bpfel.o: kernel/program.c
	@go generate -buildvcs=false && printf "$(GREEN) %s\n" "GENERATED" "$@"

kernel/program_bpfel.go: kernel/program_bpfel.o

clean:
	rm -f kernel/*.o kernel/*.go xdprobe

cleanall: clean
	rm -rf assets/
	mkdir -p assets

assets/globe.gl:
	@wget -qO $@ https://unpkg.com/globe.gl@2.45.3/dist/globe.gl.min.js && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/h3.js:
	@wget -qO $@ https://unpkg.com/h3-js@4.4.0/dist/h3-js.umd.js && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/alpine.js:
	@wget -qO $@ https://unpkg.com/alpinejs@3.15.11/dist/cdn.min.js && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/ne_110m_admin_0_countries.json:
	@wget -qO $@ https://raw.githubusercontent.com/nvkelso/natural-earth-vector/refs/heads/master/geojson/ne_110m_admin_0_countries.geojson && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/Geist-Variable.woff2:
	@wget -qO $@ https://cdn.jsdelivr.net/npm/geist@1.7.0/dist/fonts/geist-sans/Geist-Variable.woff2 && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/GeistMono-Variable.woff2:
	@wget -qO $@ https://cdn.jsdelivr.net/npm/geist@1.7.0/dist/fonts/geist-mono/GeistMono-Variable.woff2 && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/favicon.svg:
	@wget -qO $@ https://raw.githubusercontent.com/tailwindlabs/heroicons/refs/heads/master/src/24/solid/globe-europe-africa.svg && printf "$(GREEN) %s\n" "DOWNLOADED" "$@"

assets/index.css: tailwind/input.css
	@bun run build 2>/dev/null && printf "$(GREEN) %s\n" "BUILT" "$@"
