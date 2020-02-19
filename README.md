# Merlin

An agent that sends out alerts when kubernetes resources are misconfigured or have issues.

## Install

### Install CRDs
In order to install merlin, you'll need to install the CRD (custom resource definition) first, run

```bash
make install
```  
to install the CRDs



### Uninstall CRDs
To uninstall CRDs, you can run:
```bash
make uninstall
```


### Sample custom resources
Once CRDs are installed, you can start setting what rules you'd like to have, there are several samples under `config/samples`, 
you can run 
```bash
kubectl apply-samples
```
to apply all sample resources


### Install controller manager
After you've applied the custom resources, run
```bash
make deploy
``` 
to install the controller.

Note this will create a namespace called `merlin-us-dev` and install the controller manager in it.

### Debugging
You can also run the controller locally for debugging, just run 
```bash
make run
``` 


