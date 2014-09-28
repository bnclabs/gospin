Failsafe system dictionary for clustering.

- dictionary is managed as JSON.
- dictionary operations are idempotent, GetCAS(), GET(), SET(), DELETE() are
  REST compatible HEAD, GET, PUT and DELETE methods.
- JSON fields to GET(), SET() and DELETE() are specified as jsonpointer.
- system dictionary is meant to hold configuration and context information for
  the cluster.
- fault tolerance is achieved using raft protocol using go-raft
  implementations.
- can be used as a library and dictionary access APIs are natively available
  on the `Server` object.
- can be mounted on HTTP and accessed by remote client.
- can be muxed with web framework or HTTP applications using `HTTPMuxer`
  interface.
