# Merlin

An agent that sends out alerts when kubernetes resources are misconfigured or have issues.

## Install

### Install CRDs
In order to install merlin, you'll need to install the CRD (custom resource definition) first, run

```bash
make install
```  
to install the CRDs


### Adding evaluators
Once CRDs are installed, you can start setting what rules you'd like to have, there are several samples under `config/samples`, 
e.g., to install `PodEvaulator`, run 
```bash
kubectl apply -f config/samples/watcher_v1_podevaluator.yaml
```


### Install controller
After you've installed the evaluators, run
```bash
make deploy
``` 
to install the controller.


### Debugging
You can also run the controller locally for debugging, just run 
```bash
make run
``` 


