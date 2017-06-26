# k9 - a police dog to watch over DataDog

k9 is a very lightweight HTTP proxy sitting in between your [DataDog](https://www.datadoghq.com/) agent and the Datadog API, and removing the metrics and/or tags you don't really need.

## Why?

[DataDog](https://www.datadoghq.com/) is great. But pushing custom metrics to them can become pretty expensive, as [they only allow a limited number of custom metrics per host](https://help.datadoghq.com/hc/en-us/articles/204271775-What-is-a-custom-metric-and-what-is-the-limit-on-the-number-of-custom-metrics-I-can-have-).

And it's not always trivial to keep the volume of custom metrics in check:
* histograms metrics actually count as 5 different metrics by default: min, max, average, median, and 95th percentile. And [while the agent's configuration does allow cherry-picking which of those you want](https://github.com/DataDog/dd-agent/blob/5.14.1/datadog.conf.example#L103-L104), it only does that on a per-host basis: you can't say you want the 5 default metrics for some histograms, but only the average for some others ([and they don't want to make that a feature, either](https://github.com/DataDog/dd-agent/pull/3238))
* some Datadog integration libraries out there make it very easy to plug and play to for example profile your application, but they don't have hooks to cherry-pick which metrics you acutally care about
* same goes for global tags that some libraries allow using: you might not care about these for all your custom metrics

k9 aims to solve these issues by providing a very simple way to filter out the metrics and/or the tags you don't care about.

## How?

### Configuration

k9 is a very simple HTTP proxy that should sit in between the agent and Datadog's API on every host you want to use it on. It reads a very simple YML configuration file to know which metrics/tags to remove (all the fields are optional):

```yml
# should be one of DEBUG, INFO, WARN, ERROR or FATAL - defaults to INFO if not present
log_level: DEBUG

# should be a list of paths to pruning configs (see more below)
pruning_configs:
  - /opt/my_app/config/pruning_config.yml
  - /etc/k9/global_pruning_config.yml
  # also supports glob patterns
  - /etc/k9/pruning_configs/*.yml

# what port to listen on locally, defaults to 8283
listen_port: 8284

# same as the DD agent's dd_url config parameter,
# (https://github.com/DataDog/dd-agent/blob/5.14.1/datadog.conf.example#L4)
# similarly defaults to https://app.datadoghq.com
dd_url: https://my_private.datadoghq.com

```

where `pruning_configs` should be a list of paths to k9 _pruning configurations_, which should have the following shape:

```yml
metrics:
  # matching metrics will be removed altogether (except if they also match a `keep` rule)
  remove:
    - my_app.**.max
    - my_app.**.min
    - my_app.profile.**.95percentile
    - my_app.profile.**.median
    - my_app.profile.*.avg
    - a.given.metric

  # matching metrics will be kept even if they match a `remove` rule
  keep:
    - my_app.profile.some.important.function.95percentile
    - my_app.profile.some.important.function.median

tags:
  # matching metrics will have the given tags removed if present
  # (except if they also match a `keep` rule for the same tag(s))
  remove:
    - metrics:
      - my_app.**
      tags:
      - instance-type
    - metrics:
      - my_app.elasticsearch.count
      tags:
      - es_host
    - metrics:
      - '**'
      tags:
      - role

  # matching metrics will have the given tags kept if present
  # even if they also match a `remove` rule for the same tag(s)
  keep:
    - metrics:
      - my_app.hey.there
      tags:
      - instance-type

```

where double wildcards `**` will match one or more "sub-keys", e.g. `my_app.**.max` in the example above will match all of `my_app.a.max`, `my_app.a.b.max`, `my_app.a.b.c.max`, and so on; while single wildcards only match one "sub-key", e.g. `my_app.profile.*.avg` will match `my_app.profile.a.avg` but _not_ `my_app.profile.a.b.avg`.

### Running k9

Once the configuration files have been written, k9 needs simply be kept running as a service, and sent a HUP signal whenever it should re-parse the pruning configurations.

You'll also need to point your Datadog agent at your k9 instance, by adding for instance:
```yml
dd_url: http://localhost:8283
```
to your agent's configuration (see https://github.com/DataDog/dd-agent/blob/5.14.1/datadog.conf.example#L4)

### Using Chef?

If you already use [Chef](https://www.chef.io/) to manage and deploy Datadog to your hosts (presumably using [the official Datadog cookbook](https://github.com/DataDog/chef-datadog)), then deploying and using k9 is made very easy by the [k9 cookbook](https://github.com/wk8/cookbook-k9).
