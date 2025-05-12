package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const defaultTemplateText = `
# common:: переиспользуемые блоки yml
x-info:
  owner: "{{.OwnerName}} (#{{.MessangerChannel}})"
  repo: "{{.GitRepoLink}}"
x-alert-config:
  slack_alert_channel_primary: &slack_alert_channel_primary "{{.MessangerPrimaryChannel}}"
  slack_alert_mention_primary: &slack_alert_mention_primary "{{.MessangerPrimaryMention}}"
  slack_alert_channel_secondary: &slack_alert_channel_secondary "sre-slo-warnings"

_common_blocks:
  alerting_disabled_def: &alerting_disabled_def
    page_alert:
      disable: true
    ticket_alert:
      disable: true
  alerting_enabled_def: &alerting_enabled_def
    name: "ServiceSLOViolation"
    labels:
      service: "{{ "{{ $labels.sloth_service }}" }}"
      title: "{{ "{{ $labels.feature }}" }}"
      slo: "{{ "{{ $labels.sloth_slo }}" }}"
      sli_type: "{{ "{{ $labels.category }}" }}"
      source: "slo"
      alert_class: slo_violation
      alert_type: symptom
    annotations:
      doc_url: https://kb.vseinstrumenti.ru/dosearchsite.action?cql=siteSearch%20~%20%22%5C%22{{ "{{ $labels.sloth_service }}" }}-{{ "{{ $labels.sloth_slo }}" }}%5C%22%22
      grafana_dashboard_link: {{.GrafanaDashboardURL}}?from=now-6h/m&to=now-1m/m&var-service={{ "{{ $labels.sloth_service }}" }}&var-slo={{ "{{ $labels.sloth_slo }}" }}
      link: https://kb.vseinstrumenti.ru/dosearchsite.action?cql=siteSearch%20~%20%22ServiceSLOViolation%22
      runbook: docs/monitoring/slo-alerts/slo-violation-alert.md
    page_alert:
      labels:
        severity: critical
        slack_channel: *slack_alert_channel_primary
      annotations:
        slack_mention: *slack_alert_mention_primary
        summary: "Слишком высокая скорость расхода бюджета ошибок"
        description: |
          SLO: **{{ "{{ $labels.sloth_service }}" }}-{{ "{{ $labels.sloth_slo }}" }}**  \
          Высокий расход бюджета ошибок услуги - более {{.AlertWindow.Spec.Page.Quick.ErrorBudgetPercent}}% за {{.AlertWindow.Spec.Page.Quick.LongWindow}} или {{.AlertWindow.Spec.Page.Slow.ErrorBudgetPercent}}% за {{.AlertWindow.Spec.Page.Slow.LongWindow}}. Это серьезно.  \
          Проверьте состояние на графиках, если тренд не меняется более 5 минут после этого оповещения, то зарегистрируйте инцидент.
    ticket_alert:
      labels:
        severity: warning
        slack_channel: *slack_alert_channel_secondary
      annotations:
        summary: "Повышенная скорость расхода бюджета ошибок"
        description: |
           SLO: **{{ "{{ $labels.sloth_service }}" }}-{{ "{{ $labels.sloth_slo }}" }}**  \
           Расход бюджета ошибок услуги выше нормального.  \
           Израсходовано более {{.AlertWindow.Spec.Ticket.Quick.ErrorBudgetPercent}}% за {{.AlertWindow.Spec.Ticket.Quick.LongWindow}} или {{.AlertWindow.Spec.Ticket.Slow.ErrorBudgetPercent}}% за {{.AlertWindow.Spec.Ticket.Slow.LongWindow}}. Это может быть предвестником проблем. Выделите время на проверку. Проверьте графики.

# main:: основное описание SLO
version: "prometheus/v1"
service: "{{.ServiceName}}"
labels:
  title: "{{.RussianSloTitle}}"
slos:
{{- if .NeedAvailabilitySlo }}
  - name: "{{.SloName}}_availability"
    objective: 95.00
    description: |
      "SLO доступность услуги {{.RussianSloTitle}}"
    labels:
      category: availability
      slo_version: 1
    sli:
      events:
        error_query: |
          # metricQL неудовлетворительные запросы
          TODO: замените формулы на подходящие и удалите эту строку
          sum(
            increase(
              todo_replace_me_count{todo_bad_events_selector="TODO"}[{{ "{{.window}}" }}]
            ) or on() vector(0)
          )
        total_query: |
          # metricQL все запросы
          TODO: замените формулы на подходящие и удалите эту строку
          sum(
            increase(
              todo_replace_me_count{todo_total_events_selector="TODO"}[{{ "{{.window}}" }}]
            ) or on() vector(0)
          )
    alerting:
      <<: *{{if .EnableAlerts}}alerting_enabled_def{{else}}alerting_disabled_def{{end}}
{{end -}}
{{if .NeedLatencySlo}}
  - name: "{{.SloName}}_latency"
    objective: 95.00
    description: |
      "SLO время ответа услуги {{.RussianSloTitle}}"
    labels:
      category: latency
      slo_version: 1
    sli:
      raw:
        error_ratio_query: |
          # error_ratio = 1 - good/total (or bad/total)
          TODO: замените формулы на подходящие
          1 - (
            # good
            sum(
              increase(
                todo_replace_me_duration_bucket{todo_total_good_events_selector="TODO",le="todo_latency_number"}[{{ "{{.window}}" }}]
              )
            )
            / # total
            sum(
              increase(
                todo_replace_me_count{todo_total_good_events_selector="TODO"}[{{ "{{.window}}" }}]
              )
            )
          )
    alerting:
      <<: *{{if .EnableAlerts}}alerting_enabled_def{{else}}alerting_disabled_def{{end}}
{{end}}
`

type TemplateVars struct {
	OwnerName               string
	MessangerChannel        string
	GitRepoLink             string
	GrafanaDashboardURL     string
	MessangerPrimaryChannel string
	MessangerPrimaryMention string
	ServiceName             string
	RussianSloTitle         string
	SloName                 string
	NeedAvailabilitySlo     bool
	NeedLatencySlo          bool
	EnableAlerts            bool
	AlertWindow             AlertWindowSpec
}

type AlertWindowSpec struct {
	Spec struct {
		SloPeriod string `yaml:"sloPeriod"`
		Page      struct {
			Quick struct {
				ErrorBudgetPercent int    `yaml:"errorBudgetPercent"`
				ShortWindow        string `yaml:"shortWindow"`
				LongWindow         string `yaml:"longWindow"`
			} `yaml:"quick"`
			Slow struct {
				ErrorBudgetPercent int    `yaml:"errorBudgetPercent"`
				ShortWindow        string `yaml:"shortWindow"`
				LongWindow         string `yaml:"longWindow"`
			} `yaml:"slow"`
		} `yaml:"page"`
		Ticket struct {
			Quick struct {
				ErrorBudgetPercent int    `yaml:"errorBudgetPercent"`
				ShortWindow        string `yaml:"shortWindow"`
				LongWindow         string `yaml:"longWindow"`
			} `yaml:"quick"`
			Slow struct {
				ErrorBudgetPercent int    `yaml:"errorBudgetPercent"`
				ShortWindow        string `yaml:"shortWindow"`
				LongWindow         string `yaml:"longWindow"`
			} `yaml:"slow"`
		} `yaml:"ticket"`
	} `yaml:"spec"`
}

const (
	version     = "1.0.0"
	description = "This is a SLO generator program for generating SLO YAML files. It will ask you for input and generate a Sloth.dev spec as output file"
	usage       = `Usage: slogen [options]

Options:
  --help          Show this help message
  --version       Show program version
  --template      Path to custom SLO spec template file for Sloth.dev
  --show-template Show default template
  --slo-window    Path to SLO window config file`
)

func main() {
	helpFlag := flag.Bool("help", false, "Show help")
	versionFlag := flag.Bool("version", false, "Show version")
	templateFlag := flag.String("template", "", "Path to custom template file")
	showTemplateFlag := flag.Bool("show-template", false, "Show default template")
	sloWindowFile := flag.String("slo-window", "windows/google-30d.yaml", "Path to SLO window config file")
	flag.Parse()

	if *helpFlag {
		fmt.Println(description)
		fmt.Println(usage)
		return
	}

	if *versionFlag {
		fmt.Println("Version:", version)
		return
	}

	if *showTemplateFlag {
		fmt.Println("Default template:")
		fmt.Println(defaultTemplateText)
		return
	}

	var templateText string
	templateName := "default"
	if *templateFlag != "" {
		content, err := os.ReadFile(*templateFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading template file: %v\n", err)
			os.Exit(1)
		}
		templateText = string(content)
		templateName = *templateFlag
	} else {
		templateText = defaultTemplateText
	}

	alertWindowSpec, err := readSLOWindowConfig(*sloWindowFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading SLO window config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Spec template used: %s\n", templateName)
	fmt.Printf("SLO Window file: %s\n", *sloWindowFile)
	fmt.Printf("SLO Window spec: %+v\n", alertWindowSpec)

	reader := bufio.NewReader(os.Stdin)
	tmplVars := TemplateVars{}

	tmplVars.ServiceName = askRequired(reader, "Enter service name: ", "")
	tmplVars.SloName = askRequired(reader, "Enter SLO name: ", "")
	SloWindowID := askRequired(reader, "Select SLO window (1: 30d, 2: 90d)? ", "1")
	tmplVars.GrafanaDashboardURL = "https://grafana.vseinstrumenti.ru/d/slo-detail"
	if SloWindowID != "1" {
		tmplVars.GrafanaDashboardURL = "https://grafana.vseinstrumenti.ru/d/slo-detail-90d"
	}
	tmplVars.RussianSloTitle = askRequired(reader, "Enter SLO title in russian: ", "")
	tmplVars.OwnerName = askRequired(reader, "Enter SLO owner name: ", "")
	tmplVars.MessangerChannel = askRequired(reader, "Enter SLO owner messenger channel: ", "")
	tmplVars.GitRepoLink = askRequired(reader, "Enter service Git repository link: ", "")
	tmplVars.MessangerPrimaryChannel = askRequired(reader, "Enter messenger primary alerts channel: ", "")
	tmplVars.MessangerPrimaryMention = askOptional(reader, "Enter messenger primary alerts Loop mention (@group or leave empty): ", "")

	tmplVars.NeedAvailabilitySlo = askYesNo(reader, "Do you need Availability SLO (y/n)? ", true)
	tmplVars.NeedLatencySlo = askYesNo(reader, "Do you need Latency SLO (y/n)? ", true)
	tmplVars.EnableAlerts = askYesNo(reader, "Enable alerts (y/n)? ", true)
	tmplVars.AlertWindow = alertWindowSpec

	tmpl, err := template.New("slo-spec").Parse(templateText)
	if err != nil {
		panic(err)
	}

	fileName := tmplVars.SloName + ".slo.yml"
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	err = tmpl.Execute(file, tmplVars)
	if err != nil {
		panic(err)
	}

	fmt.Printf("SLO file generated: %s\n", fileName)
}

func ask(reader *bufio.Reader, question string, defaultValue string, required bool) string {
	for {
		fmt.Print(question)
		if defaultValue != "" {
			fmt.Printf("[%s] ", defaultValue)
		}
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if !required && text == "" {
			return ""
		}
		if text == "" && defaultValue != "" {
			return defaultValue
		} else if text != "" {
			return text
		}
		fmt.Println("This field cannot be empty. Please provide a value.")
	}
}
func askOptional(reader *bufio.Reader, question string, defaultValue string) string {
	return ask(reader, question, defaultValue, false)
}

func askRequired(reader *bufio.Reader, question string, defaultValue string) string {
	return ask(reader, question, defaultValue, true)
}

func askYesNo(reader *bufio.Reader, question string, defaultValue bool) bool {
	for {
		fmt.Print(question)
		if defaultValue {
			fmt.Print("[y] ")
		} else {
			fmt.Print("[n] ")
		}
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			return defaultValue
		} else if strings.ToLower(text) == "y" {
			return true
		} else if strings.ToLower(text) == "n" {
			return false
		}
		fmt.Println("Please enter 'y' or 'n'.")
	}
}

func readSLOWindowConfig(filename string) (AlertWindowSpec, error) {
	var spec AlertWindowSpec
	data, err := os.ReadFile(filename)
	if err != nil {
		return spec, err
	}
	err = yaml.Unmarshal(data, &spec)
	if err != nil {
		return spec, err
	}
	return spec, nil
}
