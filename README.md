# Kubernetes Events Collector

Events in Kubernetes cluster are stored for 1 hour by default. There are many occasions when we want to checkout why a Pod was crashed but the Events of that Pod has gone away. This component watches the Events in cluster and send then to any target like Elasticsearch or any other persistent storage which can be easily queried as you want.
