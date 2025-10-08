.DEFAULT_GOAL := help
M = $(shell printf "\033[34;1m>>\033[0m")
rwildcard = $(foreach d,$(wildcard $(1:=/*)),$(call rwildcard,$d,$2) $(filter $(subst *,%,$2),$d))

TARGET_NAME = $(firstword $(MAKECMDGOALS))
RUN_ARGS = $(filter-out $@, $(MAKEOVERRIDES) $(MAKECMDGOALS))
OS := $(shell uname -s)

SLO_WINDOW ?= 30d
OUT_GENERATED_DIR ?= .generated
OUT_PROM_RULES_DIR = $(OUT_GENERATED_DIR)/prom_rules
SLOTH_SPECS_DIR = services
SLOTH_FILE_EXT = .slo.yml
SLOTH_FILE_MASK = *$(SLOTH_FILE_EXT)
ALL_SLOS = $(call rwildcard, $(SLOTH_SPECS_DIR), $(SLOTH_FILE_MASK))
RULES_FILE_EXT = .slo.rules.yml
RULES_FILE_MASK = *$(RULES_FILE_EXT)
PROM_RULES = $(wildcard $(OUT_PROM_RULES_DIR)/$(RULES_FILE_MASK))

TOOLS_DIR = .tools/bin
SLOTH_VERSION ?= 0.11.1-vi
# Используем свою версию sloth с поддержкой проверки на дубликаты slo
SLOTH_TOOL_URL = https://github.com/vseinstrumentiru/sloth/releases/download/v$(SLOTH_VERSION)/sloth-linux-amd64
VMALERT_VERSION = 1.86.2
VMUTILS_FILENAME = vmutils-linux-amd64-v$(VMALERT_VERSION)
# GO_JSONNET_VERSION = 0.19.1
# JSONNET_BUNDLER_VERSION = 0.5.1
YQ_VERSION = 4.30.5
PROMTOOL_VERSION = 2.41.0

YQ_BINARY := $(TOOLS_DIR)/yq_$(YQ_VERSION)_$(OS)_amd64
YQ_BINARY_URL := https://github.com/mikefarah/yq/releases/download/v$(YQ_VERSION)/yq_$(OS)_amd64
# отключить генерацию alert rules для всех правил
SLOTH_DISABLE_ALERTS ?=


.PHONY: debug
debug:
	@echo ALL_SLOS: $(ALL_SLOS)
	@echo PROM_RULES: $(PROM_RULES)
	@echo "Files count in slo folders: " && find $(SLOTH_SPECS_DIR) -type f | wc -l
	@echo "slo.yml files count in slo folders: " && find $(SLOTH_SPECS_DIR) -type f -ipath '*.slo.yml' | wc -l

.PHONY: all
all: | install-tools lint build-all ## Run all tasks


.PHONY: build-all
build-all: generate ## Build all SLOTH to prometheus rules


.PHONY: lint
lint: ## Run all linters
	$(info $(M) Lint files...)
	@$(MAKE) lint-metric-queries
# TODO: add yml lint

.PHONY: lint-metric-queries
lint-metric-queries: ## Check if metric quries has unacceptable parts
	$(info $(M) Check if metric quries has unacceptable parts...)
	@FOUND_FILES=$$(grep -rEHn '[[:alpha:]]\s*[0-9]+\s*\*' services/); \
	if [ -n "$$FOUND_FILES" ]; then \
		echo "ERROR: Invalid parts in SLI formulas, it must be numbers not W*"; \
		echo "$$FOUND_FILES"; \
		exit 1; \
	else \
		echo "OK"; \
		exit 0; \
	fi

.PHONY: validate-slo-filenames
validate-slo-filenames: ## Validate SLO files names
	$(info $(M) Validate SLO files names...)
	@files=$$(find $(SLOTH_SPECS_DIR) -type f -not  -ipath '*.slo.yml'); \
	if [ -z "$$files" ]; then \
		echo "OK. All files are match *.slo.yml mask"; \
		exit 0; \
	else \
		echo "ERROR. Some unexpected files found. Files without .slo.yml mask:"; \
		echo "$$files"; \
		exit 1; \
	fi

.PHONY: validate-slo-names
validate-slo-names: ## Validate that no SLO names have service prefix
	$(info $(M) Validating SLO name don't have service prefix...)
	@errors=0; \
	for file in $$(find $(SLOTH_SPECS_DIR) -type f -name '*.slo.yml'); do \
		service=$$(grep '^service:' "$$file" | awk '{print $$2}' | tr -d '"'); \
		if grep -q -E "name:[[:space:]]+\"$$service-" "$$file"; then \
			echo "--------------------------------------------------"; \
			echo "ERROR in file: $$file"; \
			echo "Remove service name '$$service' form slo name"; \
			echo "Invalid SLO names:"; \
			grep -E "name:[[:space:]]+\"$$service-" "$$file" | sed 's/^[[:space:]]*//'; \
			errors=$$((errors+1)); \
		fi; \
	done; \
	if [ $$errors -gt 0 ]; then \
		echo "--------------------------------------------------"; \
		echo "Found $$errors files with invalid SLO names"; \
		exit 1; \
	else \
		echo "OK: No SLO names with service prefix found"; \
		exit 0; \
	fi

.PHONY: validate-sloth-specs
validate-sloth-specs: install-sloth ## Validate SLOTH spec files
	$(info $(M) Validate SLOTH spec files...)
	$(TOOLS_DIR)/sloth validate --input=$(SLOTH_SPECS_DIR) --slo-period-windows-path=./windows -e '_generated|windows'


.PHONY: validate-prom-rules
validate-prom-rules: install-vmalert ## Validate rules for vmalert
	$(info $(M) Validate rules for vmalert...)
	$(TOOLS_DIR)/vmalert -dryRun -rule="$(OUT_PROM_RULES_DIR)/$(RULES_FILE_MASK)" -rule.validateExpressions=true


.pre-generate: clean
	$(info $(M) Create a folder for Prom rules generation...)
	@mkdir -p $(OUT_PROM_RULES_DIR)
	@echo Files count to process: $$(find $(SLOTH_SPECS_DIR) -type f  -ipath '*.slo.yml' | wc -l)

.PHONY: generate
generate: | .pre-generate validate-slo-filenames validate-slo-names validate-sloth-specs $(ALL_SLOS) ## Generate SLO alert manager rules one by one


.PHONY: generate-all
generate-all: ## Generate SLO alert manager rules, with and without alerts
	 $(MAKE) generate OUT_GENERATED_DIR=.generated_with_alerts
	 $(MAKE) generate OUT_GENERATED_DIR=.generated SLOTH_DISABLE_ALERTS=true

# получиить название окна из самого файла SLO или взять значение по умолчанию SLO_WINDOW
get_slo_window = $(shell $(YQ_BINARY) e '.x-slo-window // "$(SLO_WINDOW)"' $(1))

.PHONY: $(ALL_SLOS)
$(ALL_SLOS):
	$(eval X_SLO_WINDOW := $(call get_slo_window, $@))
	$(info $(M) Generate rules for $@, slo window: '$(X_SLO_WINDOW)')
	$(TOOLS_DIR)/sloth generate --input="$@" --default-slo-period=$(X_SLO_WINDOW) --slo-period-windows-path=./windows \
		-o "$(OUT_PROM_RULES_DIR)/$(subst /,_,$(patsubst $(SLOTH_SPECS_DIR)/%$(SLOTH_FILE_EXT),%_$(X_SLO_WINDOW)$(RULES_FILE_EXT), $@))" \
		-e '$(OUT_PROM_RULES_DIR)|windows' --no-log $(if $(SLOTH_DISABLE_ALERTS),--disable-alerts)

.PHONY: promtool-check-rules
promtool-check-rules: $(PROM_RULES) ## Checks all rules files with Promtool
	$(MAKE) install-promtool
	$(info $(M) Promtool check rules...)
	@[ ! -f $(TOOLS_DIR)/promtool ] && echo ERR: No promtool util found. First run: make install-promtool || \
	 $(TOOLS_DIR)/promtool check rules --lint="all" --lint-fatal $^

.PHONY: promtool-test-alert-rules
promtool-test-alert-rules: | install-promtool $(PROM_RULES) ## Runs prometeus alert rules unit tests with Promtool
	$(info $(M) Run unit tests for generated alert rules...)
	$(TOOLS_DIR)/promtool test rules ./prom_alert_rules_tests.yml
# Какие правила тестируем указано в rule_files в prom_alert_rules_tests.yml


.PHONY: remove-alerts-descriptions
remove-alerts-descriptions: ## Remove descriptions from alerts annotations to allow run replay rules
	$(info $(M) Remove descriptions from alerts annotations...)
	find $(OUT_PROM_RULES_DIR) -type f -name "*.yml" -exec sh -c 'echo "Processing {}"; $(YQ_BINARY) eval-all '\''select(fileIndex == 0) | (.groups[].rules[] | select(has("alert"))) |= (.annotations.description = "replaced to allow rules replay (see https://docs.victoriametrics.com/vmalert.html#limitations-1)")'\'' {} -i' \;


.PHONY: check-rule-replay-can-run
check-rule-replay-can-run:
	$(info $(M) Check if we can run replay on our rules...)
	@result=$$(find $(OUT_PROM_RULES_DIR) -type f -name "*.yml" -exec $(YQ_BINARY) eval-all '.groups[].rules[] | select(has("alert") and .annotations.description | contains("query")) | length' {} \; | wc -l); \
    if [ "$$result" -eq 0 ]; then \
        echo "OK. No files contain 'query' in .annotations.description. You can run rule replay"; \
        exit 0; \
    else \
        echo "ERROR. At least one file contains 'query' in .annotations.description. You have to remove 'query' from you alert descriptions. Replay can not work with 'query' (see https://docs.victoriametrics.com/vmalert.html#limitations-1)"; \
        exit 1; \
    fi

$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

.pre-install-tools: $(TOOLS_DIR)

.PHONY: install-tools
install-tools: .pre-install-tools ## Install tools needed for development
	$(info $(M) Installing dev tools...)
	make install-yq
	make install-sloth
	make install-vmalert

install-sloth: $(TOOLS_DIR)
ifeq (,$(wildcard $(TOOLS_DIR)/sloth))
	$(info $(M) Installing sloth v$(SLOTH_VERSION)...)
	wget --show-progress -qO $(TOOLS_DIR)/sloth $(SLOTH_TOOL_URL)
	chmod +x $(TOOLS_DIR)/sloth
endif

install-vmalert: $(TOOLS_DIR)
ifeq (,$(wildcard $(TOOLS_DIR)/vmalert))
	$(info $(M) Installing vmalert v$(VMALERT_VERSION)...)
	wget --show-progress -qO- https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$(VMALERT_VERSION)/$(VMUTILS_FILENAME).tar.gz \
		| tar xvzf - -C $(TOOLS_DIR) vmalert-prod
	mv $(TOOLS_DIR)/vmalert-prod $(TOOLS_DIR)/vmalert
	chmod +x $(TOOLS_DIR)/vmalert
endif

install-yq: $(TOOLS_DIR)
ifeq (,$(wildcard $(YQ_BINARY)))
	wget -qO $(YQ_BINARY) $(YQ_BINARY_URL) --show-progress && chmod +x $(YQ_BINARY)
endif

install-promtool: $(TOOLS_DIR)
	$(info $(M) Installing promtool v$(PROMTOOL_VERSION)...)
ifeq (,$(wildcard $(TOOLS_DIR)/promtool))
	wget --show-progress -qO- "https://github.com/prometheus/prometheus/releases/download/v$(PROMTOOL_VERSION)/prometheus-$(PROMTOOL_VERSION).linux-amd64.tar.gz" \
	| tar xvzf - "prometheus-$(PROMTOOL_VERSION).linux-amd64"/promtool --strip-components=1
	mv promtool $(TOOLS_DIR)
	chmod +x $(TOOLS_DIR)/promtool
endif

.PHONY: show-slos-with-disaled-alerts
show-slos-with-disaled-alerts: ## Show SLO files with disabled alerts
	$(info $(M) Show SLO files with disabled alerts...)
	@find . -name '*.slo.yml' -type f -exec sh -c \
	'$(YQ_BINARY) e ".slos[] | select(.alerting.page_alert.disable == true) | length > 0" "$$1" | grep -q true && echo "$$1"' _ {} \;
#	@find . -name '*.slo.yml' -type f -exec $(YQ_BINARY) e '.slos[] | select(.alerting.page_alert.disable == true)' {} \; -print




.PHONY: clean
clean: ## Remove all generated files
	rm -rf $(OUT_GENERATED_DIR)/*


help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

%:
	@:
