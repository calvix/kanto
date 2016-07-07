# How Install kanto webservice
you have 3 options:
  * compile from source and run compiled binary
  * run from docker image
  * run in kubernetes

# 1. compile
prerequisites:
 * installed golang 1.5+ and git
 * permission to bind on port 80
 * GOPATH with golang packages listed in DEPS (go get package_name)
 * running kubernetes api server (url configurable via ENV "KUBERNETES_API_URL")
 
simply run commands below

 `git clone https://github.com/calvix/kanto`
 
 `cd kanto`
 
 `go build -o kanto-service`
 
 `export KUBERNETES_API_URL="kubernetes-api.server.example.com:8080"`
 
 `./kanto-service`
 
 
# 2. docker image
prerequisites:
 * running docker service
you can build your own image or use **calvix/kanto**

 `docker run --expose 80 -e KUBERNETES_API_URL=kubernetes-api.server.example.com:8080 calvix/kanto`
 
Dockerfile can be found in separate repository: https://github.com/calvix/kanto-docker 

# 3. run in kubernetes
prerequisites:
 * running kubernetes service
 
kubernetes is using docker image **calvix/kanto**

whole template is prepared in **kanto-webservice.yaml**

`kubectl create -f kanto-webservice.yaml`


**file kanto-webservice.yaml** has multiple components (deployment and service) and is using kubernetes yaml separator "**---**"
If it is not working for you (maybe older kube version) then create separate files: 1 for deployment and 1 for service and copy there corresponding parts