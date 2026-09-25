import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Bot, Play, Plus, RotateCcw, Save, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { api, Busy, Confirm, ErrorState } from "../lib";
import { useUnsavedChangesWarning } from "../UnsavedChangesBar";
import { TorrentTabs } from "./TorrentTabs";
import { PageHeader } from "../PageHeader";
import { SearchableShowPicker } from "../SearchableShowPicker";
import "../advanced-flows.css";

type Block = { id: string; type: string; config: Record<string, string> };
type Definition = { blocks: Block[] };
type Flow = { id: string; name: string; show_id?: string; revision: number; definition: Definition; created_at: number; updated_at: number };
type Event = { kind: string; show_id: string; episode_id: string; show_name: string; season: number; episode: number; triggered_at: number };
type Step = { node_id: string; status: string; input?: unknown; output?: unknown; summary?: string; duration_ms: number };
type Run = { id: string; flow_id: string; flow_revision: number; source_run_id?: string; event: Event; definition: Definition; steps: Step[]; status: string; started_at: number };
type Original = { id: string; show_name: string; season: number; episode: number; started_at: number };
type Show = { id: string; name: string; flow_id?: string };
const types = ["query.build", "jackett.search", "torrent.filter", "torrent.best", "action.download"];
const defaultLiveID = "default-live";
const defaultAnimatedID = "default-animated";
const isDefault = (id?: string) => id === defaultLiveID || id === defaultAnimatedID;
const fresh = (): Definition => ({ blocks: types.map((type, i) => ({ id: `step-${i + 1}`, type, config: {} })) });

function ChainFilterControls({ block, change }: { block: Block; change: (key: string, value: string) => void }) {
  const { t } = useTranslation();
  const numeric = [
    ["min_seeders", "torrentAutomation.minSeeders", 0, 1000000],
    ["release_delay_minutes", "torrentAutomation.releaseDelay", 1, 1440],
    ["min_mb_per_minute", "advancedFlows.minSizeRate", 1, 500],
    ["max_mb_per_minute", "advancedFlows.maxSizeRate", 1, 500],
  ] as const;
  const text = [
    ["include_keywords", "torrentAutomation.includeKeywords", false],
    ["exclude_keywords", "torrentAutomation.excludeKeywords", false],
    ["allowed_groups", "torrentAutomation.allowedGroups", true],
    ["allowed_uploaders", "torrentAutomation.allowedUploaders", true],
  ] as const;
  return <div className="advanced-chain-config">
    {numeric.map(([key, label, min, max]) => <label key={key}>{t(label)}<input type="number" min={min} max={max} value={block.config[key] ?? ""} onChange={(e) => change(key, e.target.value)} /></label>)}
    {text.map(([key, label, multiline]) => <label key={key}>{t(label)}{multiline ? <textarea className="advanced-chain-filter-list" rows={3} value={block.config[key] ?? ""} onChange={(e) => change(key, e.target.value)} /> : <input value={block.config[key] ?? ""} onChange={(e) => change(key, e.target.value)} />}</label>)}
  </div>;
}

function ChainSelectionControls({ block, change }: { block: Block; change: (key: string, value: string) => void }) {
  const { t } = useTranslation();
  return <div className="advanced-chain-config">
    <label>{t("torrentAutomation.preferredQuality")}<select value={block.config.preferred_quality ?? ""} onChange={(e) => change("preferred_quality", e.target.value)}><option value="">{t("advancedFlows.inheritSetting")}</option><option value="best">{t("torrentAutomation.bestAvailable")}</option><option value="720p">720p</option><option value="1080p">1080p</option><option value="2160p">2160p</option></select></label>
    <label>{t("torrentAutomation.preferSmaller")}<select value={block.config.prefer_smaller ?? ""} onChange={(e) => change("prefer_smaller", e.target.value)}><option value="">{t("advancedFlows.inheritSetting")}</option><option value="true">{t("common.yes")}</option><option value="false">{t("common.no")}</option></select></label>
    <label>{t("advancedFlows.maxCandidates")}<input type="number" min={1} max={20} value={block.config.max_candidates ?? ""} onChange={(e) => change("max_candidates", e.target.value)} /></label>
    {([ ["preferred_groups", "torrentAutomation.preferredGroups"], ["preferred_uploaders", "torrentAutomation.preferredUploaders"], ["preferred_providers", "torrentAutomation.preferredProviders"] ] as const).map(([key, label]) => <label key={key}>{t(label)}<textarea rows={3} value={block.config[key] ?? ""} onChange={(e) => change(key, e.target.value)} /></label>)}
  </div>;
}

export function AdvancedFlowsPage() {
  const { t } = useTranslation();
  const cache = useQueryClient();
  const [params] = useSearchParams();
  const sourceRunID = params.get("source") || "";
  const flows = useQuery<{ flows: Flow[] }>({ queryKey: ["flows", "list"], queryFn: ({ signal }) => api("/flows", "GET", undefined, signal) });
  const shows = useQuery<{ shows: Show[] }>({ queryKey: ["torrent-automation-shows"], queryFn: ({ signal }) => api("/torrents/automation/shows", "GET", undefined, signal) });
  const original = useQuery<Original>({ queryKey: ["torrent-automation-run", sourceRunID], queryFn: ({ signal }) => api(`/torrents/automation/runs/${sourceRunID}`, "GET", undefined, signal), enabled: !!sourceRunID });
  const runs = useQuery<{ runs: Run[] }>({ queryKey: ["flows", "runs"], queryFn: ({ signal }) => api("/flows/runs", "GET", undefined, signal) });
  const [current, setCurrent] = useState<Flow | null>(null);
  const [name, setName] = useState("");
  const [nameEdited, setNameEdited] = useState(false);
  const [selectedShowID, setSelectedShowID] = useState("");
  const [copySourceID, setCopySourceID] = useState("");
  const [definition, setDefinition] = useState<Definition>(fresh);
  const [event, setEvent] = useState({ kind: "show_available", show_name: "", season: 1, episode: 1, runtime_minutes: 0, media_profile: "" });
  const [run, setRun] = useState<Run | null>(null);
  const [recorded, setRecorded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  useEffect(() => { if (!current && flows.data?.flows.length) select(flows.data.flows[0]); }, [flows.data]);
  const dirty = !!current && (name !== current.name || JSON.stringify(definition) !== JSON.stringify(current.definition) || (!current.id && (!!selectedShowID || !!copySourceID)));
  useUnsavedChangesWarning(dirty, busy, t("common.unsavedNavigation"));
  const display = recorded && run ? run.definition : definition;
  const visibleRun = run && (recorded || JSON.stringify(definition) === JSON.stringify(run.definition)) ? run : null;
  const choose = (next: Flow) => { if (dirty && !window.confirm(t("common.unsavedNavigation"))) return; select(next); };
  function select(flow: Flow) { setCurrent(flow); setName(flow.name); setNameEdited(false); setSelectedShowID(flow.show_id || ""); setCopySourceID(""); setEvent((old) => ({ ...old, show_name: shows.data?.shows.find((show) => show.id === flow.show_id)?.name || "" })); setDefinition(structuredClone(flow.definition)); setRun(null); setRecorded(false); setFeedback(""); }
  function create() { if (dirty && !window.confirm(t("common.unsavedNavigation"))) return; select({ id: "", name: t("advancedFlows.newName"), revision: 0, definition: fresh(), created_at: 0, updated_at: 0 }); }
  function config(index: number, key: string, value: string) { setDefinition((old) => ({ blocks: old.blocks.map((block, i) => i === index ? { ...block, config: { ...block.config, [key]: value } } : block) })); }
  async function save() {
    if (!current) return;
    setBusy(true);
    try {
      const saved: Flow = await api(current.id ? `/flows/${current.id}` : "/flows", current.id ? "PUT" : "POST", { name, show_id: current.show_id || "", revision: current.revision, definition });
      setCurrent(saved); setName(saved.name); setNameEdited(false); setCopySourceID(""); setDefinition(structuredClone(saved.definition));
      await cache.invalidateQueries({ queryKey: ["flows", "list"] });
      const linkShow = selectedShowID && shows.data?.shows.find((show) => show.id === selectedShowID)?.flow_id !== saved.id && !isDefault(saved.id);
      if (linkShow) {
        await api(`/torrents/automation/shows/${selectedShowID}/chain`, "PUT", { flow_id: saved.id });
        await cache.invalidateQueries({ queryKey: ["torrent-automation-shows"] });
      }
      setFeedback(linkShow ? t("advancedFlows.savedAndLinked", { name: shows.data?.shows.find((show) => show.id === selectedShowID)?.name || selectedShowID }) : t("advancedFlows.saved"));
    } catch (error) { setFeedback(String(error)); } finally { setBusy(false); }
  }
  async function assignShow() {
    if (!current?.id || !selectedShowID || dirty) return;
    setBusy(true);
    try {
      await api(`/torrents/automation/shows/${selectedShowID}/chain`, "PUT", { flow_id: current.id });
      await cache.invalidateQueries({ queryKey: ["torrent-automation-shows"] });
      setFeedback(t("advancedFlows.assignedToShow", { name: shows.data?.shows.find((show) => show.id === selectedShowID)?.name || selectedShowID }));
    } catch (error) { setFeedback(String(error)); } finally { setBusy(false); }
  }
  async function remove() {
    if (!current?.id) return;
    await api(`/flows/${current.id}`, "DELETE");
    setCurrent(null); setRun(null); setConfirmDelete(false);
    await Promise.all([cache.invalidateQueries({ queryKey: ["flows", "list"] }), cache.invalidateQueries({ queryKey: ["flows", "runs"] }), cache.invalidateQueries({ queryKey: ["torrent-automation-shows"] })]);
    setFeedback(t("advancedFlows.flowDeleted"));
  }
  async function resetDefault() {
    if (!current?.id || !isDefault(current.id)) return;
    setBusy(true);
    try {
      const saved: Flow = await api(`/flows/${current.id}/reset`, "POST", { revision: current.revision });
      select(saved);
      await cache.invalidateQueries({ queryKey: ["flows", "list"] });
      setFeedback(t("advancedFlows.defaultReset"));
    } catch (error) { setFeedback(String(error)); } finally { setBusy(false); setConfirmReset(false); }
  }
  async function test() {
    if (!current?.id) return;
    setBusy(true);
    try {
      const result: Run = await api(`/flows/${current.id}/replay`, "POST", { ...(sourceRunID ? { source_run_id: sourceRunID } : { event }), definition: dirty ? definition : undefined });
      setRun(result); setRecorded(false);
      await cache.invalidateQueries({ queryKey: ["flows", "runs"] });
      setFeedback(t("advancedFlows.testComplete"));
    } catch (error) { setFeedback(String(error)); } finally { setBusy(false); }
  }
  async function openRun(id: string) {
    if (dirty && !window.confirm(t("common.unsavedNavigation"))) return;
    try {
      const result: Run = await api(`/flows/runs/${id}`);
      const selected = flows.data?.flows.find((item) => item.id === result.flow_id);
      if (selected) select(selected);
      setRun(result); setRecorded(true);
    } catch (error) { setFeedback(String(error)); }
  }
  if (flows.error || shows.error) return <ErrorState error={flows.error || shows.error || new Error("Unavailable")} retry={() => { void flows.refetch(); void shows.refetch(); }} />;
  if (!flows.data || !shows.data) return <Busy />;
  const scope = current?.show_id ? shows.data.shows.find((show) => show.id === current.show_id)?.name || current.show_id : isDefault(current?.id) ? t("advancedFlows.profileDefaultHelp") : t("advancedFlows.chainGeneral");
  const selectedShow = shows.data.shows.find((show) => show.id === selectedShowID);
  const eventShowName = sourceRunID ? original.data?.show_name : event.show_name;
  return <div className="page advanced-flows-page">
    <PageHeader eyebrow={t("search.eyebrow")} title={t("search.title")} description={t("advancedFlows.chainDescription")} />
    <TorrentTabs />
    <div className="advanced-chain-layout">
      <section className="panel advanced-flow-setup" aria-label={t("advancedFlows.flow")}>
        <button type="button" className="button" onClick={() => create()}><Plus size={16} />{t("advancedFlows.newPreset")}</button>
        <label>{t("advancedFlows.savedFlows")}
          <select value={current?.id || ""} onChange={(e) => { const flow = flows.data.flows.find((item) => item.id === e.target.value); if (flow) choose(flow); }}><option value="">{current && !current.id ? t("advancedFlows.newPreset") : t("advancedFlows.chooseFlow")}</option>{flows.data.flows.map((flow) => <option key={flow.id} value={flow.id}>{flow.id === defaultLiveID ? t("advancedFlows.defaultLive") : flow.id === defaultAnimatedID ? t("advancedFlows.defaultAnimated") : flow.name}{flow.show_id ? ` · ${shows.data.shows.find((show) => show.id === flow.show_id)?.name || flow.show_id}` : ""}</option>)}</select>
        </label>
        {current && <><p className="muted">{scope}</p><label>{t("advancedFlows.name")}<input value={current.id === defaultLiveID ? t("advancedFlows.defaultLive") : current.id === defaultAnimatedID ? t("advancedFlows.defaultAnimated") : name} maxLength={100} disabled={isDefault(current.id)} onChange={(e) => { setName(e.target.value); setNameEdited(true); }} /></label>
          {!current.id && <label>{t("advancedFlows.copyFromChain")}<select value={copySourceID} onChange={(e) => { const id = e.target.value; setCopySourceID(id); const sourceFlow = flows.data.flows.find((flow) => flow.id === id); setDefinition(sourceFlow ? structuredClone(sourceFlow.definition) : fresh()); }}><option value="">{t("advancedFlows.chooseChainToCopy")}</option>{flows.data.flows.map((flow) => <option key={flow.id} value={flow.id}>{flow.id === defaultLiveID ? t("advancedFlows.defaultLive") : flow.id === defaultAnimatedID ? t("advancedFlows.defaultAnimated") : flow.name}{flow.show_id ? ` · ${shows.data.shows.find((show) => show.id === flow.show_id)?.name || flow.show_id}` : ""}</option>)}</select></label>}
          {!isDefault(current.id) && <div className="advanced-show-assignment">
            <SearchableShowPicker shows={shows.data.shows.filter((show) => !current.show_id || show.id === current.show_id)} value={selectedShowID} onChange={(id) => { setSelectedShowID(id); const selected = shows.data.shows.find((show) => show.id === id); setEvent((old) => ({ ...old, show_name: selected?.name || "" })); if (!current.id && !nameEdited) setName(selected?.name || current.name); }} disabled={busy} />
            {!!current.id && <button type="button" className="button" disabled={busy || dirty || !selectedShowID || selectedShow?.flow_id === current.id} onClick={() => void assignShow()}>{selectedShow?.flow_id === current.id ? t("advancedFlows.alreadyAssigned") : t("advancedFlows.assignToShow")}</button>}
          </div>}
          <button type="button" className="button primary" disabled={busy || !dirty} onClick={() => void save()}><Save size={16} />{selectedShowID && (!current.id || selectedShow?.flow_id !== current.id) && !isDefault(current.id) ? t("advancedFlows.saveAndLink") : t("advancedFlows.saveChanges")}</button>
          {isDefault(current.id) && <button type="button" className="button" disabled={busy} onClick={() => setConfirmReset(true)}><RotateCcw size={16} />{t("advancedFlows.resetDefault")}</button>}
          {!!current.id && !isDefault(current.id) && <button type="button" className="button danger" disabled={busy} onClick={() => setConfirmDelete(true)}><Trash2 size={16} />{t("advancedFlows.deleteFlow")}</button>}
          <div className="advanced-event-setup"><h3 className="advanced-testing-title"><Play size={16} />{t("advancedFlows.testing")}</h3>{sourceRunID ? <p className="muted">{original.data ? t("advancedFlows.historicalSource", { name: original.data.show_name, season: original.data.season, episode: original.data.episode }) : t("advancedFlows.loadingHistoricalSource")}</p> : <div className="advanced-custom-event"><label>{t("advancedFlows.showName")}<input value={event.show_name} maxLength={200} onChange={(e) => setEvent({ ...event, show_name: e.target.value })} /></label><label>{t("advancedFlows.season")}<input type="number" min={0} max={100} value={event.season} onChange={(e) => setEvent({ ...event, season: Number(e.target.value) })} /></label><label>{t("advancedFlows.episode")}<input type="number" min={1} max={1000} value={event.episode} onChange={(e) => setEvent({ ...event, episode: Number(e.target.value) })} /></label><label>{t("advancedFlows.runtimeMinutes")}<input type="number" min={0} max={600} value={event.runtime_minutes} onChange={(e) => setEvent({ ...event, runtime_minutes: Number(e.target.value) })} /></label><label>{t("advancedFlows.mediaProfile")}<select value={event.media_profile} onChange={(e) => setEvent({ ...event, media_profile: e.target.value })}><option value="">{t("advancedFlows.testProfileDefault")}</option><option value="live">{t("advancedFlows.defaultLive")}</option><option value="animated">{t("advancedFlows.defaultAnimated")}</option></select></label></div>}
          {original.error && <ErrorState error={original.error} retry={() => void original.refetch()} />}
          <button type="button" className="button" disabled={busy || !current.id || (sourceRunID ? !original.data : !event.show_name.trim())} onClick={() => void test()}><Play size={16} />{t("advancedFlows.testEvent")}</button><small>{t("advancedFlows.chainTestHelp")}</small>
          </div></>}
      </section>
      <section className="advanced-chain" aria-label={t("advancedFlows.chainTitle")}>
        <div className="advanced-chain-event"><Bot size={18} /><strong>{selectedShow ? t("advancedFlows.chainEventForShow", { name: selectedShow.name }) : t("advancedFlows.chainEvent")}</strong>{!selectedShow && eventShowName && <span>{eventShowName}</span>}</div>
        {current ? display.blocks.map((block, i) => { const step = visibleRun?.steps.find((item) => item.node_id === block.id); return <div key={block.id} className="advanced-chain-stage-wrap"><div className="advanced-chain-line" /><article className={`panel advanced-chain-stage advanced-node-${step?.status || "waiting"}`}><div className="advanced-chain-stage-heading"><span className="badge">{i + 1}</span><div><h2>{t(`advancedFlows.nodes.${block.type.replace(".", "_")}`)}</h2><p>{t(`advancedFlows.chainStage.${block.type.replace(".", "_")}`)}</p></div>{step && <span className="badge">{t(`advancedFlows.status.${step.status}`, { defaultValue: step.status })}</span>}</div>
          {!recorded && i === 0 && <div className="advanced-chain-config"><label>{t("advancedFlows.config.titleOverride")}<input value={block.config.title_override || ""} maxLength={200} onChange={(e) => config(i, "title_override", e.target.value)} /></label><label>{t("advancedFlows.config.prefix")}<input value={block.config.prefix || ""} maxLength={200} onChange={(e) => config(i, "prefix", e.target.value)} /></label><label>{t("advancedFlows.config.suffix")}<input value={block.config.suffix || ""} maxLength={200} onChange={(e) => config(i, "suffix", e.target.value)} /></label></div>}
          {!recorded && i === 2 && <ChainFilterControls block={block} change={(key, value) => config(i, key, value)} />}
          {!recorded && i === 3 && <ChainSelectionControls block={block} change={(key, value) => config(i, key, value)} />}
          {step && <details className="advanced-chain-result"><summary>{step.summary || t("advancedFlows.testRun")}</summary>{step.input !== undefined && <div><strong>{t("advancedFlows.input")}</strong><pre>{JSON.stringify(step.input, null, 2)}</pre></div>}{step.output !== undefined && <div><strong>{t("advancedFlows.output")}</strong><pre>{JSON.stringify(step.output, null, 2)}</pre></div>}</details>}
        </article></div>; }) : <p className="muted">{t("advancedFlows.chainEmpty")}</p>}
      </section>
      <section className="panel advanced-history"><h2>{t("advancedFlows.runs")}</h2>{runs.data?.runs.length ? runs.data.runs.map((item) => <button className="button" type="button" key={item.id} onClick={() => void openRun(item.id)}>{item.event.show_name} · {t(`advancedFlows.status.${item.status}`, { defaultValue: item.status })}</button>) : <p>{t("advancedFlows.noRuns")}</p>}{recorded && <button type="button" className="button" onClick={() => setRecorded(false)}>{t("advancedFlows.editCurrent")}</button>}</section>
    </div>
    {run && !visibleRun && <p className="advanced-feedback">{t("advancedFlows.chainChanged")}</p>}
    {feedback && <p role="status" className="advanced-feedback">{feedback}</p>}
    {confirmDelete && current && <Confirm title={t("advancedFlows.deleteFlowTitle")} message={t("advancedFlows.deleteFlowMessage", { name: current.name })} onClose={() => setConfirmDelete(false)} onConfirm={remove} />}
    {confirmReset && current && <Confirm title={t("advancedFlows.resetDefault")} message={t("advancedFlows.resetDefaultConfirm")} onClose={() => setConfirmReset(false)} onConfirm={resetDefault} />}
  </div>;
}
