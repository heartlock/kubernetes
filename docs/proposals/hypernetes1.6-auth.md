# Proposal of hypernetes1.6 auth
- Continue to use keystone, Keystone authentication is enabled by passing the --experimental-keystone-url=<AuthURL> option to the API server during startup

- Use  [RBAC](https://kubernetes.io/docs/admin/authorization/) for authorization，enable the authorization module with --authorization-mode=RBAC

- Add auth-controller to manage RBAC policy

## auth workflow:
```
kubectl                 apiserver                keystone               rbac         auth+controller
   +                       +                         +                    +                 +
   |         1             |                         |                    |                 |
   +-----+request+--------->                         |                    |                 |
   |                       |           2             |                    <--+update policy++
   |                       +----+Authentication+----->                    |                 |
   |                       |           3             |                    |                 |
   |                       <---+user.info,success+---+                    |                 |
   |                       |                      4  +                    |                 |
   |                       +-------------------+Authorization+------------>                 |
   |                       |                     5   +                    |                 |
   |                       <--------------------+success+-----------------+                 |
   |          6            |                         +                    |                 |
   <-----+response+--------+                         |                    |                 |
   |                       |                         |                    |                 |
   +                       +                         +                    +                 +

```

- To update rbac policy, auth-controller need to get user-info. Auth-controller update `ClusterRole` and `RoleBinding` to get permissions of namespaces, when user is updated. May be we should add user as a kind of resource.

- Watch `Namespace` and update the permissions of namespace scoped resources for regular user.

1. kubectl send request with (username, password) to apiserver
2. Apiserver receive request from kubectl, and call [AuthenticatePassword](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/plugin/pkg/authenticator/password/keystone/keystone.go#L42)(username, password) to check (username, password) via keystone. 
3. If check successfully return (user.info, true), continue to step4. if check fail return (user.info, fales), request fail.
4. call [Authorize](https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/auth/authorizer/rbac/rbac.go#L45)(authorizer.Attributes) to check RBAC roles  and binding. 
5. if check success return true and execute operation of the request. if check fail return false ,request fail.
6. return request result 

## Add user
- Use Third Party Resources to create user resource:
```
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
metadata:
  name: user.stable.example.com
description: "A specification of a User"
versions:
- name: v1
```

- Add a user object. May be there more detail user information will be set using custom fields like `username`. At least username are required.
```
apiVersion: "stable.example.com/v1"
kind: User
metadata:
  name: alice
username: "alice chen"
```



## auth-controller 

auth-controller is responsible of updating RBAC roles and binding when

- new user is created: allow the user to create/get namespace
- new namespace is created/deleted
- network is created/deleted
- user is removed: delete all user related data

auth -controller will watch resources including `User`, `NameSpace`, `Network` and update RBAC roles and binding using superuser.
First  create `ClusterRole` for regular user to get permissions of `NameSpace` :
```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced.
  name: namespace-creater
rules:
  - apiGroups: [""]
    resources: ["namespace"]
    verbs: ["get", "create"] # I think verbs should include "delete"，user have permission of deleting their namespace 
```

### new user are created:
create `ClusterRoleBinding` for new regular user reference `namespace-creater` role:

```
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  name: username-namespace-creater
subjects:
  - kind: User 
    name: username
roleRef:
  kind: ClusterRole
  name: namespace-creater
  apiGroup: rbac.authorization.k8s.io
```
### a new namespace are created:
create `Role` namespaced:
```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  namespace: namespace
  name: access-resources-within-namespace
rules:
  - apiGroups: [""]
    resources: [ResourceAll]
    verbs: [VerbAll]
```
create `Rolebinding` reference to `role` namespace:
```
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1alpha1
metadata:
  name: namespace-binding
  namespace: namespace 
subjects:
  - kind: User
    name: username
roleRef:
  kind: Role
  name: access-resources-within-namespace
  apiGroup: rbac.authorization.k8s.io
```
role and rolebinding will be removed when namespace are deleted
### a network is created/deleted
`Network` will be defined non-namespaced resource. we can set permission of `Network` in `ClusterRole` when user are created.

### a user is deleted:
`ClusterRoleBinding` related the user will be removed at least. 
 
