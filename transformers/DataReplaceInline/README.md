# DataReplaceInline kustomize plugin
**Notes:** [Read](https://github.alm.europe.cloudcenter.corp/ccc-paas/kustomize-plugins/blob/main/README.md) the generic documentation to understand kustomize plugins

## Introduction
*DataReplaceInline* is an exec plugin of transformer's family written in Go.

## Placement
The plugin is placed within ArgoCD custom docker image on the path:
~~~
 /home/argocd/.config/kustomize/plugin/transformers.kustomize.com/v1/datareplaceinline/DataReplaceInline
~~~

## Configuration
A kustomization.yaml file could have the following example lines. The configuration of the plugin are the lines 7 and 8:
~~~
1 apiVersion: kustomize.config.k8s.io/v1beta1
2 kind: Kustomization
3 resources:
4   - example-local-secret.yaml
5   - github.com/example/kustomize/bases?ref=v1.0.6
6
7 transformers:
8   - dataReplaceInline.yaml
~~~
Given this, the kustomization process would expect to find a file called *dataReplaceInline.yaml* in the kustomization root.

This is the plugin’s configuration file. It contains a YAML configuration object.

The file *dataReplaceInline.yaml* must contain:
~~~
1 apiVersion: transformers.kustomize.com/v1
2 kind: DataReplaceInline
3 metadata:
4   name: datareplaceinline
5 gitFileConf: "../cluster.ini"
~~~
This file is the plugin configuration file. Kustomize executes the plugin providing this file as the first parameter, so, as developer, you can set the YAML fields you need for the correct functioning of the plugin.

In this case, the line 5 is the custom YAML field which the plugins needs, the path for the git configuration file.

Lines 1 and 2 describe where the plugin is stored, see [Read](https://github.alm.europe.cloudcenter.corp/ccc-paas/kustomize-plugins/blob/main/README.md) for more details.

Line 4 is the name. You can use whatever name you want, but if you configure several plugins, you must make sure that the name is unique.

## How it works
The plugin read from **stdin** all resources sended from kustomize and process line by line searching Vault and Git patterns. If it finds any, the pattern is replaced with the correct value (see Patterns). In short, the plugin does not process YAML, but text

## Patterns
The plugin searches two types of patterns. One of the for data from Vault and the other one for data from Git

Pattern for Vault:
~~~
${vault:vault-secret@vault-key}
~~~
Example:
~~~
baseDN: "${vault:caas/data/sync-ldap@baseDN}"
~~~
In this example, the plugin must replace "${vault:caas/data/sync-ldap@baseDN}" for the right value stored in Vault. In this case "caas/data/sync-ldap" is the name of the Vault secret and "baseDN" is the name of the key within the secret

#### **Important:**
The plugin uses VAULT_HOST and VAULT_TOKEN environment variables for lines with data from Vault. Those variables are set in the custom ArgoCD docker image.

Pattern for Git:
~~~
${git:variable}
~~~
Example:
~~~
  url: ldaps://${git:LDAP_URL}:636/${git:LDAP_FILTER}
~~~
In this example, the plugin must replace "${git:LDAP_URL}" and "${git:LDAP_FILTER}" patterns for the right values stored in a git file. The git file is indicated in the YAML file configuration, specifically in the "gitFileConf" field. It must be in ini format without any sections.

Example cluster.ini file:
~~~
LDAP_BIND_DN=uid=openshift,ou=example
LDAP_URL=ldap.example.org
LDAP_FILTER=o=example?corpAliasLocal?sub?(&(objectClass=inetOrgPerson))
~~~
It also can be used a Vault pattern as a Git value.

Example:
~~~
LDAP_USER="${vault:caas/data/sync-ldap@ldapUser}"
~~~
In this example the data is in the git file, but the value point to a Vault secret.

## Modifiers
In this plugin some modifiers have been developed to modify the result.

There are two types of modifiers:
- Data modifiers, they modify the data
- Line modifiers, they modify the entire line

Data modifiers:
~~~
- base64
- select(regex)
- dict(key)
- default(value)
~~~
Line modifiers:
~~~
- indentN
~~~
The way to use them is as follows (you can see the examples below):
~~~
${pattern | modifier}
~~~

### base64
---
This modifier encodes the value in base64

### select(regex)
---
This modifier selects the value from a list.

**Notes:** The value between () is a regex.

Example git file conf cluster.ini:
~~~
AWS_REGIONS="eu-west-1,eu-west-2,eu-west-3"
~~~
Then, you can use a configuration as follows:
~~~
awsRegion: ${git:AWS_REGIONS | select(^\w+-\w+-2$)}
~~~
Or in this case simpler:
~~~
awsRegion: ${git:AWS_REGIONS | select(2)}
~~~

### dict(key)
---
This modifier selects the value from a list of dictionaries.

Example git file cluster.ini:
~~~
aws_private_subnets=zone-a=subnet-085c8042ddabd63ee,zone-b=subnet-060c12e675d9bcfec,zone-c=subnet-0514acf4477d0c0d4
~~~
Then, you can use a configuration as follows:
~~~
awsSubnet: ${git:aws_private_subnets | dict(zone-c)}
~~~

### default(value)
---
This modifier sets the default value in case it does not exist.
~~~
awsRegion: ${git:AWS_REGIONS | default(eu-west-3)}
~~~

### indentN
---
This modifier indent N spaces the whole line from the beginning.
~~~
cert: |
   ${ vault:caas/data/certs@ca-bundle.crt | indent2 }
~~~

## Annotations
- Kustomize uses standard kubernetes manifests. Everything that is not a standard manifest is considered as a plugin, so if you have to configure CR's you have to use the List kind

OK:
~~~
apiVersion: v1
kind: List
items:
- apiVersion: machine.openshift.io/v1beta1
  kind: MachineSet
  metadata:
    creationTimestamp: null
    labels:
      machine.openshift.io/cluster-api-cluster: ${git:clustername}
    name: worker-1a
    namespace: openshift-machine-api
  spec:
    ....
~~~

Wrong:
~~~
apiVersion: machine.openshift.io/v1beta1
kind: MachineSet
metadata:
  creationTimestamp: null
  labels:
    machine.openshift.io/cluster-api-cluster: ${git:clustername}
  name: worker-1a
  namespace: openshift-machine-api
spec:
  ....
~~~

In this case, kustomize tries to find a plugin in ~/.config/kustomize/plugin/machine.openshift.io/v1beta1/machineset/{MachineSet.so,MachineSet}

## Examples
More examples in the [examples](https://github.alm.europe.cloudcenter.corp/ccc-paas/kustomize-plugins/blob/main/transformers/datatreplaceinline/examples) folder

## Debugging
This plugin is a go binary, so when is in develop phase you can use delve to debug the code.

Remember you must provide the file configuration as first parameter and the YAML manifests for the standard input.
~~~
dlv debug DataReplaceInline.go -r secret.yaml -- file-conf.yaml
~~~
You can also run the plugin from bash to see how it works
~~~
go build -o DataReplaceInline DataReplaceInline.go
cat openshift-manifests.yaml | ./DataReplaceInline file-conf.yaml
~~~
To see how it works within the ArgoCD pod:
- Open a remote shell into the container
- Clone the kustomize repo
- Run kustomize with the parameter --enable_alpha_plugins

If everything works fine, you will see all the manifests rendered with the values ​​correctly replaced by the plugin
~~~
oc get pods -n argocd
oc rsh argocd-repo-server-xxxxx bash

argocd-repo-server-xxxxx$ cd /tmp
argocd-repo-server-xxxxx/tmp$ git clone github.com/example/kustomize-resources.git && cd kustomize-resources
argocd-repo-server-xxxxx/tmp/kustomize-resources$ kustomize build --enable_alpha_plugins .
~~~
You can also run the plugin within the pod
~~~
oc get pods -n argocd
oc rsh argocd-repo-server-xxxxx bash

argocd-repo-server-xxxxx$ cd /tmp
argocd-repo-server-xxxxx/tmp$ git clone github.com/example/kustomize-resources.git && cd kustomize-resources
argocd-repo-server-xxxxx/tmp/kustomize-resources$ cat configmap.yaml | \
  ~argocd/.config/kustomize/plugin/transformers.kustomize.com/v1/datareplaceinline/DataReplaceInline/DataReplaceInline file-conf.yaml
~~~

## Logging
If something goes wrong, the plugin creates a log file within the pod in /tmp/DataReplaceInline.log
