# Janitor
Janitor is a general purpose *Proxy* with additional abilities like *Service Discovery* and *Load Balance*. It was designed to expose services with multiple tasks directly to end user or behind other tool like Nginx, HAProxy or even H5.

For the time present Janitor can load a `upstream` with multiple
`targets` that stored in Consul and expose it with a specified port on the very host that Janitor was deployed.

# Features

  * Automaticlly bind or release port for a service base on
    information that store in Consul
  * Proxy HTTP requests to backend upstreams
  * Load balance requests with Round Robin

# Installation
 
```
  make 
```

## Consul
  
  * how to define a upstream in consul, following is the service struct
    stored in Consul

 ```

 curl -XGET yourconsuladdr:8500/v1/catalog/service/mesos

 {
        "Address": "192.168.1.103",
        "Node": "foobar",
        "ServiceAddress": "192.168.1.103",
        "ServiceID": "mesos",
        "ServiceName": "mesos",
        "ServicePort": 5100,
        "ServiceTags": [
            "borg",
            "borg-frontend-proto:http",
            "borg-frontend-port:3412"
        ]
    }
```

make sure ServiceTags follow the conventions

  * `borg` indicates current service is a valid upstream
  * `borg-frontend-proto:http` the protocol of this upstream
  * `borg-frontend-port:3412` the port number of this upstream

# Concepts

  * `upstream` or more commonly a `Service`, is a tuple of
    {ServiceName, Port, Proto}

  * `Target` upstream have multiple targets, underlaying is a mesos
    tasks.

# Author

  cming.xu@gmail.com

# How to Contribute?

  Fork it and make a PR

