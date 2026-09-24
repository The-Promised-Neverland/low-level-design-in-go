"use client";
import { useCallback, useEffect, useState } from "react";
import type { Ev, Robot, State, Vehicle } from "@/lib/types";
import s from "./page.module.css";

const API = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";
const STRATEGIES = ["NEAREST", "LOAD_BALANCED", "ZONE_BALANCED", "COMPACTION"];
const TYPE_LABEL: Record<string, string> = { BIKE: "Bike", CAR: "Car", TRUCK: "Truck" };
const zoneColor = (z: string) => (z === "TRUCK" ? "var(--truck)" : z === "BIKE" ? "var(--bike)" : z === "CAR" ? "var(--car)" : "var(--dim)");
const floorLabel = (n: number | null | undefined) => {
  if (n == null) return "?";
  if (n < 0) return "bot";
  return `F${n}`;
};

async function getJSON<T>(path: string): Promise<T> {
  const r = await fetch(API + path, { cache: "no-store" });
  if (!r.ok) throw new Error(`${path} ${r.status}`);
  return r.json();
}

function Chip({ v, robotFloor }: { v: Pick<Vehicle, "vehicleRegNo" | "vehicleType"> & Partial<Vehicle>; robotFloor?: string }) {
  const cls = [s.chip, v.unparkRequested ? s.leaving : "", v.parkingStatus === "REARRANGING" ? s.moving : ""].join(" ");
  return (
    <span className={cls} style={{ borderColor: zoneColor(v.vehicleType) }} title={`${TYPE_LABEL[v.vehicleType]} ${v.vehicleRegNo}`}>
      <b style={{ color: zoneColor(v.vehicleType) }}>{v.vehicleType[0]}</b>{v.vehicleRegNo}
      {v.unparkRequested && <i> exit</i>}{v.parkingStatus === "REARRANGING" && <i> moving</i>}{robotFloor && <i> {robotFloor}</i>}
    </span>
  );
}

function RobotRow({ r }: { r: Robot }) {
  const busy = r.status === "BUSY";
  const move = r.type === "PARKING" ? `Staging → ${floorLabel(r.toFloor)}` : r.type === "UNPARKING" ? `${floorLabel(r.fromFloor)} → Exit` : `${floorLabel(r.fromFloor)} → ${floorLabel(r.toFloor)}`;
  return (
    <li className={`${s.robot} ${busy ? s.busy : ""}`}>
      <span className={s.rid}>{r.id}</span>
      <span className={s.rstate}>{busy ? "Busy" : "Idle"}</span>
      <span className={s.rmove}>{busy ? <>{r.vehicleRegNo} · {move}</> : "Waiting for a job"}</span>
    </li>
  );
}

export default function Page() {
  const [st, setSt] = useState<State | null>(null);
  const [events, setEvents] = useState<Ev[]>([]);
  const [down, setDown] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const refresh = useCallback(async () => {
    try {
      await getJSON("/api/health");
      const [a, b] = await Promise.all([getJSON<State>("/api/state"), getJSON<Ev[]>("/api/events").catch(() => null)]);
      setSt(a); if (b) setEvents(b.slice(-50)); setDown(null);
    } catch (e) { setDown(e instanceof Error ? e.message : "unreachable"); }
  }, []);
  useEffect(() => { refresh(); const t = setInterval(refresh, 700); return () => clearInterval(t); }, [refresh]);
  useEffect(() => { if (toast) { const t = setTimeout(() => setToast(null), 4000); return () => clearTimeout(t); } }, [toast]);

  async function pick(strategy: string) {
    setPending(true);
    try {
      const r = await fetch(API + "/api/strategy", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ strategy }) });
      if (!r.ok) { const j = await r.json().catch(() => ({})); setToast(`Strategy not changed: ${j.error ?? r.status}`); }
      await refresh();
    } catch { setToast("Strategy not changed: backend unreachable"); }
    setPending(false);
  }

  if (!st) return <main className={s.center}><h1>Parking control room</h1><p>{down ? `Can't reach the backend at ${API} (${down}). Start it and this page will connect on its own.` : "Connecting…"}</p></main>;

  const robots = [...st.robots.parking, ...st.robots.unparking, ...st.robots.shuffle];
  const floors = [...st.floors].sort((a, b) => b.floorNumber - a.floorNumber);
  const inbound = (f: number) => robots.filter(r => r.status === "BUSY" && r.toFloor === f && r.type !== "UNPARKING");
  const outbound = (f: number) => robots.filter(r => r.status === "BUSY" && r.fromFloor === f && r.type !== "PARKING");
  const m = st.metrics;
  const busyParking = st.robots.parking.filter(r => r.status === "BUSY");

  return (
    <main className={s.page}>
      {down && <div className={s.offline}>Backend not responding ({down}). Showing the last snapshot.</div>}
      {toast && <div className={s.toast} role="alert">{toast}</div>}
      <header className={s.top}>
        <h1>Parking control room</h1>
        <div className={s.seg} role="group" aria-label="Parking strategy">
          {STRATEGIES.map(x => (
            <button key={x} aria-pressed={st.strategy === x} className={st.strategy === x ? s.on : ""} disabled={pending || st.shufflingInProgress} onClick={() => pick(x)}>{x.replace("_", " ").toLowerCase()}</button>
          ))}
        </div>
      </header>
      {st.shufflingInProgress && <div className={s.shuffle}>Rearrangement in progress. Shuffle robots are moving vehicles between floors; strategy changes are paused.</div>}
      <section className={s.metrics}>
        {([["Arrived", m.vehiclesArrived], ["Accepted", m.vehiclesAccepted], ["Rejected", m.vehiclesRejected], ["Exited", m.vehiclesExited], ["Active", m.activeVehicles]] as const).map(([k, v]) => (
          <div key={k}><strong>{v}</strong><span>{k}</span></div>))}
      </section>

      <div className={s.grid}>
        <section>
          <h2>Entrance queue <small>{st.queues.staging.length} waiting for a parking robot</small></h2>
          <div className={s.staging}>
            {st.queues.staging.length === 0 && <p className={s.dim}>No vehicles waiting.</p>}
            {st.queues.staging.map(q => <Chip key={q.ticketId} v={{ vehicleRegNo: q.vehicleRegNo, vehicleType: q.vehicleType as Vehicle["vehicleType"] }} robotFloor={`→ F${q.toFloor}`} />)}
            {busyParking.length > 0 && <p className={s.flow}>{busyParking.length} parking robot{busyParking.length > 1 ? "s" : ""} carrying vehicles from staging to floors</p>}
          </div>

          <h2>Floors <small>top of stack = highest floor</small></h2>
          <div className={s.floors}>
            {floors.map(f => {
              const free = Math.max(0, f.capacity - Math.max(f.reservedCapacity, f.physicallyUsedCapacity));
              const pct = (n: number) => `${(n / Math.max(1, f.capacity)) * 100}%`;
              const compaction = f.capacity > 0 ? Math.round((f.reservedCapacity / f.capacity) * 1000) / 10 : 0;
              const inn = inbound(f.floorNumber), out = outbound(f.floorNumber);
              const vs = f.vehicles ?? st.vehicles.filter(v => v.physicalFloor === f.floorNumber);
              return (
                <div key={f.floorNumber} className={s.floor}>
                  <div className={s.fhead}>
                    <span className={s.fnum}>{f.floorNumber}</span>
                    <span className={s.compact}>{compaction}% compact</span>
                    <span className={s.dim}>{f.physicallyUsedCapacity} parked · {f.reservedCapacity} reserved · {free} free of {f.capacity}</span>
                    {inn.map(r => <span key={r.id} className={s.arrive}>↓ {r.vehicleRegNo} arriving ({r.id})</span>)}
                    {out.map(r => <span key={r.id} className={s.depart}>↑ {r.vehicleRegNo} leaving ({r.id}){r.type === "SHUFFLE" ? ` to ${floorLabel(r.toFloor)}` : " to exit"}</span>)}
                  </div>
                  <div className={s.bar} title="parked / reserved / free">
                    <i style={{ width: pct(f.physicallyUsedCapacity), background: "var(--text)" }} />
                    <i style={{ width: pct(Math.max(0, f.reservedCapacity - f.physicallyUsedCapacity)), background: "var(--amber)", opacity: .55 }} />
                  </div>
                  <div className={s.vlist}>{vs.map(v => <Chip key={v.ticketId} v={v} />)}</div>
                </div>
              );
            })}
          </div>
        </section>

        <aside>
          <h2>Robots</h2>
          {([["Parking", st.robots.parking], ["Unparking", st.robots.unparking], ["Shuffle", st.robots.shuffle]] as const).map(([n, list]) => (
            <div key={n}><h3>{n}</h3><ul className={s.robots}>{list.map(r => <RobotRow key={r.id} r={r} />)}</ul></div>))}
          <h2>Waiting <small>exit {st.queues.waitingForUnpark.length} · shuffle {st.queues.waitingForShuffle.length}</small></h2>
          <div className={s.staging}>
            {st.queues.waitingForUnpark.map(q => <Chip key={q.ticketId} v={{ vehicleRegNo: q.vehicleRegNo, vehicleType: q.vehicleType as Vehicle["vehicleType"] }} robotFloor={`${floorLabel(q.fromFloor)} → exit`} />)}
            {st.queues.waitingForShuffle.map(q => <Chip key={q.ticketId} v={{ vehicleRegNo: q.vehicleRegNo, vehicleType: q.vehicleType as Vehicle["vehicleType"] }} robotFloor={`${floorLabel(q.fromFloor)} → ${floorLabel(q.toFloor)}`} />)}
          </div>
          <h2>Event log</h2>
          <ol className={s.log}>
            {[...events].reverse().map((e, i) => (
              <li key={i}><time>{new Date(e.timestamp).toLocaleTimeString()}</time> <b>{e.type.replaceAll("_", " ").toLowerCase()}</b> {e.message ?? [e.vehicleRegNo, e.robotId, e.fromFloor != null ? floorLabel(e.fromFloor) : "", e.toFloor != null ? `→ ${floorLabel(e.toFloor)}` : ""].filter(Boolean).join(" ")}</li>))}
          </ol>
        </aside>
      </div>

      <section>
        <h2>All vehicles <small>{st.vehicles.length}</small></h2>
        <div className={s.tablewrap}><table className={s.table}>
          <thead><tr><th>Ticket</th><th>Reg</th><th>Type</th><th>Status</th><th>On floor</th><th>Target</th><th>Exit requested</th></tr></thead>
          <tbody>{st.vehicles.map(v => (
            <tr key={v.ticketId}><td>{v.ticketId}</td><td>{v.vehicleRegNo}</td><td>{TYPE_LABEL[v.vehicleType]}</td><td>{v.parkingStatus.replace("_", " ").toLowerCase()}</td><td>{floorLabel(v.physicalFloor)}</td><td>{floorLabel(v.targetFloor)}</td><td>{v.unparkRequested ? "yes" : ""}</td></tr>))}</tbody>
        </table></div>
      </section>
    </main>
  );
}
