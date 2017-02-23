# multi-tenancy architecture
## plan for auth
- use namespace as tenant(one-to-one mapping)
- create network use TPR(namespace-tenant one-to-one mapping) 
- auth-controller is responsible for creating/deleting user(tenant in hypernetes) in keystone and updating RBAC when namespace created/deleted
- network-controller powers network as before

## reasons of using this plan
- Simple implementation
- no permission leakage

## shortcoming
- architecture is flat. Tenant only have one namespace and network.
- network can not insulate in tenant
