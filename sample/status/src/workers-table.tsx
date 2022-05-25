import { Component, h } from "preact";

import type { WorkersByRole } from "./model";
import { ThreadLoadStat } from "./thread-load-stat";

interface Props {
  workers: WorkersByRole;
}

export class WorkersTable extends Component<Props> {
  override render() {
    const workerEntries = Object.entries(this.props.workers).sort(([a], [b]) => a.localeCompare(b));
    return (
      <table class="pure-table pure-table-horizontal">
        <thead>
          <tr>
            <th>role</th>
            <th>#</th>
            <th>NUMA</th>
            <th>load</th>
          </tr>
        </thead>
        <tbody>
          {workerEntries.map(([role, workers]) => workers.sort((a, b) => a.nid - b.nid).map((w, i) => (
            <tr key={w.id}>
              {i === 0 ? (
                <td rowSpan={workers.length}>{role}</td>
              ) : undefined}
              <td title={w.id}>{w.nid}</td>
              <td>{w.numaSocket}</td>
              <td style="text-align: right;"><ThreadLoadStat id={w.id}/></td>
            </tr>
          )))}
        </tbody>
      </table>
    );
  }
}