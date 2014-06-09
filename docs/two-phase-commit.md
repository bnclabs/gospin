## two-phase commit for co-ordination failures

**false positive**, is a scenario when client thinks that an update request
succeeded but the system is not yet updated, due to failures.

**false-negative**, is a scenario when client thinks that an update request has
failed but the system has applied the update into StateContext.

Meanwhile, when system is going through a rebalance or a kv-rollback or
executing a new DDL statement, Index-Coordinator can crash. But this situation
should not put the system in in-consistent state. To achieve this, we propose a
solution with the help of ns-server.

* Index-Coordinator and its replica shall maintain a persisted copy of its
  local StateContext.
* ns-server will act as commit-point-site.
* for every update, Index-Coordinator will increment the CAS and create a
  transaction tuple {CAS, status} with ns-server.
* status will be "initial" when it is created and later move to either "commit"
  or "rollback".
* for every update accepted by Index-Coordinator it shall create a new copy of
  StateContext, persist the new copy locally as log and push the copy to its
  replicas.
  * if one of the replica fails, Index-Coordinator will post "rollback" status
    to ns-server for this transaction and issue a rollback to each of the
    replica.
  * upon rollback, Index-Coordinator and its replica will delete the local
    StateContext log and return failure.
* when all the replicas accept the new copy of StateContext and persist it
  locally as a log, Index-Coordinator will post "commit" status to
  ns-server for this transaction and issue commit to each of the replica.
* during commit, local log will be switched as the new StateContext, and the
  log file will be deleted.

whenever a master or replica fails it will always go through a bootstrap phase
during which it shall detect a log of StateContext, consult with ns-server
for the corresponding transaction's status. If status is "commit" it will switch
the log as new StateContext, otherwise it shall delete the log and retain the
current StateContext.

* once ns-server creates a new transaction entry, it should not accept a new
  IndexManager into the cluster until the transaction moves to "commit" or
  "rollback" status.
* during a master election ns-server shall pick a master only from the set of
  IndexManagers that took part in the last-transaction.
* ns-server should maintain a rolling log of transaction status.

if ns-server allows new IndexManager to join the cluster, then false-positive is
possible when master crashes immediately and a new node, that did not
participate in the previous update, becomes a new master. This can be avoided
if master locks ns-server to prevent any new nodes from joining the cluster
until the update is completed.
