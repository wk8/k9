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

k9 is a very simple HTTP proxy that should sit in between the agent and Datadog's API on every host you want to use it on. It reads a very simple configuration file to know which metrics/tags to remove:

```yml
# should be one of DEBUG, INFO, WARN, ERROR or FATAL - defaults to INFO if not present
log_level: DEBUG

# should be a list of paths to pruning configs (see more below)
pruning_configs:
  - /opt/my_app/config/pruning_config.yml
  - /etc/k9/global_pruning_config.yml

# what port to listen on locally, defaults to 8283
listen_port: 8284

# same as the DD agent's dd_url config parameter,
# (https://github.com/DataDog/dd-agent/blob/5.14.1/datadog.conf.example#L4)
# similarly defaults to https://app.datadoghq.com
dd_url: https://my_private.datadoghq.com

```


