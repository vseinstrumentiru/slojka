# Каталог SLI/SLO

**Это демо проект для демонстрации структуры и инструментов. Можно использовать как основу своего каталога SLO.**

Этот каталог содержит определеня SLI/SLO для услуг. Сгруппировано по сервисам.

Каждый файл содержит описание одного SLO, одного или нескольких SLI в формате [sloth.dev для cli](https://sloth.dev/specs/default/).
Из этих файлов описаний при помощи [sloth.dev cli generate](https://sloth.dev/usage/cli/) для prometheus (или victoriametrics) генерируются правила записи ([recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/)),
определения оповещений ([alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)).


# Разработка 

## Размещение файлов

Для каждого технического сервиса создаем отдельный каталог (тут сервис = приложение), внутри помещаем файл определения SLI/SLO выбранные для этапа пути пользователя в этом сервисе.

Формат имени файла: `<название этапа латинскими символами>.slo.yml`

## Генерация файлов правил записи для prometheus

Установить необходимые для сборки утилиты:

    make install-tools

Сгенерировать файлы правил:

    SLO_WINDOW=7d make all

или

    SLO_WINDOW=30d make all

где 7d и 30d это интервал наблюдения для SLO (возможно настроить [дополнительные окна SLO](https://sloth.dev/usage/slo-period-windows/)).
Итервал наблюдения можно указать только при генерации, так как внутри файлов определений используется шаблон {{.window}} для удобства и однообразия при генерации.
По умолчаню доступны окна 7 и 30 дней. Чтобы добавить собственные окна создайте описание соглсно [документации](https://sloth.dev/usage/slo-period-windows/#custom-slo-period-catalog).

### Генерация правил подсчета без alert rules

    SLOTH_DISABLE_ALERTS=true make generate

или

    SLOTH_DISABLE_ALERTS=true make all

## Проверка правильности сгенерированных файлов

При генерации sloth проверяет правильность спецификации slo и правильность PromQL выражений `expr` и прерывает сборку, если они содержат ошибку.

См. также [Prometheus: Syntax-checking rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#syntax-checking-rules)

# Работа со спецификациями sloth.dev

Мы применяем yaml anchors (якоря) для переиспользования частей yml файла, потому в встретите якоря вида `slack_alert_channel_primary: &slack_alert_channel_primary`, где & обозначает начало имени якоря. И использование значения из якоря `slack_channel: *slack_alert_channel_primary`, где * означает вставку значения якоря с указанным именем. Якоря могут содержать как один так и множество вложенных узлов, например, это мы как раз и используем, чтобы не повторять код определения алертов `&alerting_def`.

## Дополнительные поля

Мы добавили нестандартные для sloth спецификации поля в yml, чтобы облегчить управление SLO.

```yaml
    x-info:
        owner: "Группа по автоматизации торговли"
        repo: https://git/myshop
    x-alert-config: # настройки алертов: основной канал для критических (primary) и второй (secondary) для менее важных уведомлений
        slack_alert_channel_primary: &slack_alert_channel_primary slo-monitoring # использовать в alerting -> page_alert -> labels
        slack_alert_mention_primary: &slack_alert_mention_primary @group_name или в Slack <!subteam^S90d0303> # использовать в alerting -> page_alert -> annotations.slack_mention
        slack_alert_channel_secondary: &slack_alert_channel_secondary slo-warnings # использовать в alerting -> ticket_alert -> labels
        slack_alert_mention_secondary: &slack_alert_mention_secondary @group_name или в Slack <!subteam^S90d0404>  # использовать в alerting -> ticket_alert -> annotations.slack_mention
    x-slo-window: 90d
```
где,

  * `x-info` - группа метданных: владелец, репозитарий сервиса, первичная документация SLO
  * `x-alert-config` - настройки алертов: каналы slack `slack_alert_channel_*` для критических (primary) и второй (secondary) для менее важных уведомлений, и упоминания `slack_alert_mention_*` для первичного и вторичного канала (если оставить пустым, то упоменанеия не будет)
  * `x-slo-window` - если нужно указать отличное от обычного окна SLO. Лучше всегда указывать явно. Например: 30d или 90d. Доступные занчения в slos-catalog/sloth/windows поле sloPeriod

Они специально вынесены в верх файла, чтобы было возможно быстро найти эти важные данные.

## Наши label для метрики SLI

Обязательно добавить в общие label:

```yaml
    labels:
      title: "Добавить в корзину"
```
где,

* `title` - название на русском для SLI, чтобы было понятно продакт-менеджерам, продакт-оунерам, бизнесу что это за этап/шаг.

Для каждого SLO добавить label:

```yaml
    labels:
        slo_version: 1
```
* `slo_version` - версия формул подсчета и цели, мы увеличиваем slo_version при изменении цели, бизнес-процесса. Так мы на графиках можем видеть, что изменения применились. Также это приводит к сбросу бюджета ошибок, он начинает считаться снова от 100%. Если мы сильно изменили формулы подсчета (правили ошибки), то возможно также нужно увеличить версию, так как бюджет уже может быть израсходован по неверным показаниям. Если наоборот не было расхода бюджета (100% всегда) и изменили формулы подсчета - версию поднять.

## Настройка алертов для SLO

### Общие метки алетов

Для всех алертов мы используем один общий набор меток, в которых часть значений - шаблонны для alermanager (запролняются в момент создания алерта):
```yaml
    labels:
        service: "{{ $labels.sloth_service }}"
        title: "{{ $labels.title }}"
        slo: "{{ $labels.sloth_slo }}"
        sli_type: "{{ $labels.category }}"
        source: "slo"
        alert_class: slo_violation
        alert_type: symptom
```
###  Служебные метки алертов

* `doc_url` - ссылка для связи между метрикой и первичным документом SLO (сформированный id slo позволяет найти документ в Confluence, учтите вы должны быть залогинены в Confluence)
* `grafana_dashboard_link` - позваляет создавать ссылку напрямую на показатели данного SLO на доске [Детализация SLO](https://grafana/d/slo-detail) иди  [Детализация SLO 90d](https://grafana/d/slo-detail-90d)
* `link` - ссылка на документ описывающий данные тип алерта ServiceSLOViolation
* `runbook` - ссылка на общий документ, что делать с этим алертом (постоянная)
```yaml
    annotations:
        doc_url: https://confluence/dosearchsite.action?cql=siteSearch%20~%20%22%5C%22{{ $labels.sloth_service }}-{{ $labels.sloth_slo }}%5C%22%22
        grafana_dashboard_id: slo-detail/slo-detalizatsiia
        grafana_dashboard_link: https://grafana/d/slo-detail?from=now-6h/m&to=now-1m/m&var-service={{ $labels.sloth_service }}&var-slo={{ $labels.sloth_slo }}
        link: https://confluence/dosearchsite.action?cql=siteSearch%20~%20%22ServiceSLOViolation%22
        runbook: docs/monitoring/slo-alerts/slo-violation-alert.md
```

### Разделение алертов по каналам

Вы можете отправляеть алерты в 2 разные канала для page и ticket алетртов, также можно упоминать группы или пользователей (пример, для группы в loop просто упомниние @группа, в slack `<!subteam^GROUP_ID>` см https://api.slack.com/reference/surfaces/formatting#mentioning-groups).
Формирование текста алерта для Slack происходит согласно нашему [шаблону для AlertManager](https://github.com/vseinstrumentiru/slojka/blob/main/alertmanger/notification_template.tmpl), если у вас неправильно формируется алерт, то проверьте совпадение полей в шаблоне и в файле спецификации sloth (возможно поле оказалось в labels, вместо annotations).

Пример настройки критического алерта (page) в канал с упоминанием. Если упоминание не нужно, то просто оставьте пустое значение или удалите строку с slack_mention.
И предупреждение без упоминания (ticket).

```yaml
     x-alert-config: # настройки алертов: основной канал для критических (primary) и второй (secondary) для менее важных уведомлений
        slack_alert_channel_primary: &slack_alert_channel_primary slo-monitoring
        slack_alert_mention_primary: &slack_alert_mention_primary <!subteam^S90d0303>
        slack_alert_channel_secondary: &slack_alert_channel_secondary sre-slo-warnings
        slack_alert_mention_secondary: &slack_alert_mention_secondary ""
...
    page_alert:
        labels:
          severity: critical
          slack_channel: *slack_alert_channel_primary
        annotations
          ...
          slack_mention: *slack_alert_mention_primary

    ticket_alert:
        labels:
          severity: warning
          slack_channel: *slack_alert_channel_secondary
        annotations:
          ...
        slack_mention: *slack_alert_mention_secondary
```

