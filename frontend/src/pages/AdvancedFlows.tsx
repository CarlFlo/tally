import {
  Background,
  Controls,
  Handle,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge,
  type Node,
  type NodeProps,
  type XYPosition,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  FlaskConical,
  GripVertical,
  Plus,
  Save,
  Trash2,
  Workflow,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { api, Busy, Confirm, dateLabel, ErrorState } from "../lib";
import { useUnsavedChangesWarning } from "../UnsavedChangesBar";
import { TorrentTabs } from "./TorrentTabs";
import "../advanced-flows.css";

type Port = { name: string; type: string };
type Kind = { type: string; inputs: Port[]; outputs: Port[]; config: string[]; input_mode?: "one"; deprecated?: boolean };
type FlowNode = {
  id: string;
  type: string;
  config: Record<string, string>;
  x: number;
  y: number;
};
type FlowEdge = {
  id: string;
  source: string;
  source_port: string;
  target: string;
  target_port: string;
};
type Definition = { nodes: FlowNode[]; edges: FlowEdge[] };
type Flow = {
  id: string;
  name: string;
  revision: number;
  definition: Definition;
  created_at: number;
  updated_at: number;
};
type Step = {
  node_id: string;
  status: string;
  input?: unknown;
  output?: unknown;
  summary?: string;
  duration_ms: number;
};
type Run = {
  id: string;
  flow_id: string;
  flow_revision: number;
  source_run_id?: string;
  event: {
    kind: string;
    show_id: string;
    episode_id: string;
    show_name: string;
    season: number;
    episode: number;
    triggered_at: number;
  };
  definition: Definition;
  steps: Step[];
  status: string;
  started_at: number;
  duration_ms: number;
};
type RunSummary = Omit<Run, "definition" | "steps">;
type HistoricalRun = {
  id: string;
  show_id: string;
  episode_id: string;
  show_name: string;
  season: number;
  episode: number;
  status: string;
  started_at: number;
  query: string;
  decision_log: Array<{ stage: string; status: string; summary: string }>;
};
type CustomEvent = {
  kind: "show_available";
  show_name: string;
  season: number;
  episode: number;
};
type Preview = { input?: unknown; output?: unknown; note?: string; previousInput?: boolean };

function previewFor(
  id: string,
  graph: Definition,
  event: Record<string, unknown> | null,
  previous: Run | null,
  visited = new Set<string>(),
): Preview {
  const node = graph.nodes.find((item) => item.id === id);
  if (!node || visited.has(id)) return { note: "needsInput" };
  visited.add(id);
  if (node.type.startsWith("trigger."))
    return event ? { output: event } : { note: "chooseEvent" };
  const edge = graph.edges.find((item) => item.target === id);
  const upstream: Preview = edge ? previewFor(edge.source, graph, event, previous, visited) : {};
  const previousInput = previous?.steps.find((step) => step.node_id === id)?.input;
  const input = upstream.output ?? previousInput;
  const fromPrevious = upstream.output === undefined && previousInput !== undefined;
  if (input === undefined) return { note: upstream.note || (edge ? "runTest" : "needsInput") };
  if (node.type === "query.build") {
    const source = input as Record<string, unknown>;
    if (typeof source.show_name !== "string" || typeof source.season !== "number" || typeof source.episode !== "number")
      return { input, previousInput: fromPrevious, note: "runTest" };
    const episode = `S${String(source.season).padStart(2, "0")}E${String(source.episode).padStart(2, "0")}`;
    const parts = [node.config.prefix?.trim(), source.show_name, episode, node.config.suffix?.trim()].filter(Boolean);
    return { input, output: parts.join(" "), previousInput: fromPrevious };
  }
  if (node.type === "text.replace") {
    const key = node.config.attribute || (typeof input === "string" ? "value" : edge?.target_port === "candidate" ? "name" : "show_name");
    if (!node.config.find) return { input, previousInput: fromPrevious, note: "findText" };
    if (typeof input === "string") {
      if (key !== "value") return { input, previousInput: fromPrevious, note: "chooseField" };
      const replaced = input.replaceAll(node.config.find, node.config.replace || "");
      return { input, output: node.config.trim !== "false" ? replaced.trim() : replaced, previousInput: fromPrevious };
    }
    if (!input || typeof input !== "object" || typeof (input as Record<string, unknown>)[key] !== "string")
      return { input, previousInput: fromPrevious, note: "chooseField" };
    const output = { ...(input as Record<string, unknown>) };
    const replaced = (output[key] as string).replaceAll(node.config.find, node.config.replace || "");
    output[key] = node.config.trim !== "false" ? replaced.trim() : replaced;
    return { input, output, previousInput: fromPrevious };
  }
  if (node.type === "action.log" || node.type === "action.stop" || node.type === "action.download")
    return { input, previousInput: fromPrevious, note: "terminal" };
  return { input, previousInput: fromPrevious, note: "runTest" };
}
type CanvasData = { kind: Kind; status?: string; label: string } & Record<
  string,
  unknown
>;
type CanvasNode = Node<CanvasData, "flow">;

function FlowCard({ data }: NodeProps<CanvasNode>) {
  const { t } = useTranslation();
  return (
    <div className={`advanced-node advanced-node-${data.status || "waiting"}`}>
      {data.kind.inputs.map((port) => (
        <div className={`advanced-port-row input port-${port.type}`} key={port.name}>
          <Handle
            type="target"
            position={Position.Left}
            id={port.name}
            style={{ top: "50%" }}
          />
          <small>
            <span className="advanced-port-direction">{t("advancedFlows.inputPort")}</span>
            {t(`advancedFlows.ports.${port.name}`)}
          </small>
        </div>
      ))}
      <div className="advanced-node-heading">
        <strong>{data.label}</strong>
        {data.status && data.status !== "waiting" && (
          <span className="advanced-node-status">
            {t(`advancedFlows.status.${data.status}`)}
          </span>
        )}
      </div>
      {data.kind.outputs.map((port) => (
        <div className={`advanced-port-row output port-${port.type}`} key={port.name}>
          <small>
            <span className="advanced-port-direction">{t("advancedFlows.outputPort")}</span>
            {t(`advancedFlows.ports.${port.name}`)}
          </small>
          <Handle
            type="source"
            position={Position.Right}
            id={port.name}
            style={{ top: "50%" }}
          />
        </div>
      ))}
    </div>
  );
}
const nodeTypes = { flow: FlowCard };
const nodeDragType = "application/x-tally-flow-node";
const starter: Definition = {
  nodes: [
    { id: "trigger", type: "trigger.episode", config: {}, x: 0, y: 140 },
    { id: "query", type: "query.build", config: {}, x: 250, y: 140 },
    { id: "search", type: "jackett.search", config: {}, x: 500, y: 140 },
    { id: "filter", type: "torrent.filter", config: { min_seeders: "1" }, x: 750, y: 140 },
    { id: "best", type: "torrent.best", config: {}, x: 1000, y: 140 },
    { id: "download", type: "action.download", config: {}, x: 1250, y: 140 },
  ],
  edges: [
    {
      id: "trigger-query",
      source: "trigger",
      source_port: "event",
      target: "query",
      target_port: "event",
    },
    {
      id: "query-search",
      source: "query",
      source_port: "query",
      target: "search",
      target_port: "query",
    },
    {
      id: "search-filter",
      source: "search",
      source_port: "results",
      target: "filter",
      target_port: "results",
    },
    {
      id: "filter-best",
      source: "filter",
      source_port: "results",
      target: "best",
      target_port: "results",
    },
    {
      id: "best-download",
      source: "best",
      source_port: "candidate",
      target: "download",
      target_port: "candidate",
    },
  ],
};

export function AdvancedFlowsPage() {
  const { t } = useTranslation();
  const statusLabel = (status: string) =>
    t(`advancedFlows.status.${status}`, {
      defaultValue: status.replaceAll("_", " "),
    });
  const cache = useQueryClient();
  const [params] = useSearchParams();
  const kinds = useQuery<{ nodes: Kind[] }>({
    queryKey: ["flows", "kinds"],
    queryFn: ({ signal }) => api("/flows/nodes", "GET", undefined, signal),
  });
  const flows = useQuery<{ flows: Flow[] }>({
    queryKey: ["flows", "list"],
    queryFn: ({ signal }) => api("/flows", "GET", undefined, signal),
  });
  const history = useQuery<{ runs: HistoricalRun[] }>({
    queryKey: ["torrent-automation-runs"],
    queryFn: ({ signal }) =>
      api("/torrents/automation/runs?limit=100", "GET", undefined, signal),
  });
  const runs = useQuery<{ runs: RunSummary[] }>({
    queryKey: ["flows", "runs"],
    queryFn: ({ signal }) => api("/flows/runs", "GET", undefined, signal),
  });
  const [current, setCurrent] = useState<Flow | null>(null);
  const [name, setName] = useState("");
  const [nodes, setNodes, onNodesChange] = useNodesState<CanvasNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selected, setSelected] = useState("");
  const [selectedEdge, setSelectedEdge] = useState("");
  const [sourceID, setSourceID] = useState(params.get("source") || "");
  const [customEvent, setCustomEvent] = useState<CustomEvent>({
    kind: "show_available",
    show_name: "",
    season: 1,
    episode: 1,
  });
  const [inspection, setInspection] = useState<Run | null>(null);
  const [visibleSteps, setVisibleSteps] = useState(0);
  const [recorded, setRecorded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [canvasKey, setCanvasKey] = useState(0);
  const [canvasDropActive, setCanvasDropActive] = useState(false);
  const [confirmFlowDelete, setConfirmFlowDelete] = useState(false);
  const screenToFlowPosition = useRef<((position: XYPosition) => XYPosition) | null>(null);
  const kindMap = useMemo(
    () => new Map(kinds.data?.nodes.map((kind) => [kind.type, kind]) || []),
    [kinds.data],
  );
  const label = (type: string) =>
    t(`advancedFlows.nodes.${type.replace(".", "_")}`, {
      defaultValue: type.replace(".", " "),
    });
  const toCanvas = (definition: Definition) => {
    setNodes(
      definition.nodes.map((node) => ({
        id: node.id,
        type: "flow",
        position: { x: node.x, y: node.y },
        data: {
          kind: kindMap.get(node.type)!,
          label: label(node.type),
          config: node.config,
        },
      })),
    );
    setEdges(
      definition.edges.map((edge) => ({
        id: edge.id,
        source: edge.source,
        sourceHandle: edge.source_port,
        target: edge.target,
        targetHandle: edge.target_port,
      })),
    );
  };
  const definition = (): Definition => ({
    nodes: nodes.map((node) => ({
      id: node.id,
      type: node.data.kind.type,
      config: node.data.config as Record<string, string>,
      x: node.position.x,
      y: node.position.y,
    })),
    edges: edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      source_port: edge.sourceHandle || "",
      target: edge.target,
      target_port: edge.targetHandle || "",
    })),
  });
  const selectFlow = (flow: Flow) => {
    setCanvasKey((key) => key + 1);
    setCurrent(flow);
    setName(flow.name);
    toCanvas(flow.definition);
    setInspection(null);
    setRecorded(false);
    setSelected("");
    setSelectedEdge("");
    setFeedback("");
  };
  useEffect(() => {
    if (!current && kinds.data && flows.data?.flows.length)
      selectFlow(flows.data.flows[0]);
  }, [kinds.data, flows.data]);
  const hasChanges =
    !!current &&
    (!current.id ||
      name !== current.name ||
      JSON.stringify(definition()) !== JSON.stringify(current.definition));
  const discardAllowed = () =>
    !hasChanges || window.confirm(t("common.unsavedNavigation"));
  const newFlow = () => {
    if (!discardAllowed()) return;
    const flow: Flow = {
      id: "",
      name: t("advancedFlows.newName"),
      revision: 0,
      definition: starter,
      created_at: 0,
      updated_at: 0,
    };
    selectFlow(flow);
  };
  useUnsavedChangesWarning(hasChanges, busy, t("common.unsavedNavigation"));
  const original = history.data?.runs.find((run) => run.id === sourceID);
  const customReady =
    customEvent.show_name.trim().length > 0 &&
    customEvent.show_name.length <= 200 &&
    Number.isInteger(customEvent.season) &&
    customEvent.season >= 0 &&
    customEvent.season <= 100 &&
    Number.isInteger(customEvent.episode) &&
    customEvent.episode >= 1 &&
    customEvent.episode <= 1000;
  useEffect(() => {
    if (!inspection || visibleSteps >= inspection.steps.length) return;
    const timer = window.setTimeout(
      () => setVisibleSteps((count) => count + 1),
      220,
    );
    return () => window.clearTimeout(timer);
  }, [inspection?.id, visibleSteps]);
  const traceMatches =
    !inspection ||
    recorded ||
    JSON.stringify(definition()) === JSON.stringify(inspection.definition);
  const steps = traceMatches
    ? inspection?.steps.slice(0, visibleSteps) || []
    : [];
  const runningID = inspection?.steps[visibleSteps]?.node_id;
  const selectedStep = steps.find((step) => step.node_id === selected);
  const statusFor = (id: string) =>
    steps.find((step) => step.node_id === id)?.status ||
    (runningID === id ? "running" : "waiting");
  const activeNodes =
    recorded && inspection
      ? inspection.definition.nodes.map((node): CanvasNode => ({
          id: node.id,
          type: "flow",
          position: { x: node.x, y: node.y },
          data: {
            kind: kindMap.get(node.type)!,
            label: label(node.type),
            status: statusFor(node.id),
          },
        }))
      : nodes.map((node) => ({
          ...node,
          data: {
            ...node.data,
            label: label(node.data.kind.type),
            status: statusFor(node.id),
        },
      }));
  const selectedNodeType =
    activeNodes.find((node) => node.id === selected)?.data.kind.type || "";
  const rejected =
    activeNodes.find((node) => node.id === selected)?.data.kind.type ===
    "torrent.filter"
      ? (selectedStep?.output as
          | {
              rejection_counts?: Record<string, number>;
              rejected?: Array<{ name: string; reasons: string[] }>;
            }
          | undefined)
      : undefined;
  const reasonLabel = (code: string) =>
    code === "below_min_seeders"
      ? t("torrentRuns.filterSeeders")
      : code === "confidence_too_low"
        ? t("torrentRuns.filterConfidence")
        : t(`torrentConfidence.reason.${code}`, {
            defaultValue: code.replaceAll("_", " "),
          });
  const graphEdges =
    recorded && inspection
      ? inspection.definition.edges.map((edge): Edge => ({
          id: edge.id,
          source: edge.source,
          sourceHandle: edge.source_port,
          target: edge.target,
          targetHandle: edge.target_port,
        }))
      : edges;
  const selectedInputEdge = graphEdges.find((edge) => edge.target === selected);
  const selectedInputType = activeNodes
    .find((node) => node.id === selected)
    ?.data.kind.inputs.find((port) => port.name === selectedInputEdge?.targetHandle)?.type;
  const previewEvent: Record<string, unknown> | null = sourceID === "custom"
    ? customReady ? { ...customEvent } : null
    : original ? {
        kind: "episode_released",
        source_run_id: original.id,
        show_id: original.show_id,
        episode_id: original.episode_id,
        show_name: original.show_name,
        season: original.season,
        episode: original.episode,
        triggered_at: original.started_at,
      } : inspection ? { ...inspection.event } : null;
  const selectedPreview = selected && !recorded
    ? previewFor(selected, definition(), previewEvent, inspection)
    : null;
  const followed = new Set(
    steps
      .filter((step) => step.status !== "skipped")
      .map((step) => step.node_id),
  );
  const activeEdges = graphEdges.map((edge) => ({
    ...edge,
    style:
      followed.has(edge.source) && followed.has(edge.target)
        ? { stroke: "#4ab890", strokeWidth: 3 }
        : undefined,
  }));
  const compatible = (connection: Connection | Edge) => {
    const from = activeNodes.find((node) => node.id === connection.source)?.data
      .kind;
    const to = activeNodes.find((node) => node.id === connection.target)?.data
      .kind;
    const output = from?.outputs.find(
      (port) => port.name === connection.sourceHandle,
    );
    const input = to?.inputs.find(
      (port) => port.name === connection.targetHandle,
    );
    const replacementInput = activeEdges.find((edge) => edge.target === connection.source);
    const replacementInputType = from?.type === "text.replace"
      ? from.inputs.find((port) => port.name === replacementInput?.targetHandle)?.type
      : undefined;
    const replacementOutputMismatch = to?.type === "text.replace" && activeEdges.some((edge) => {
      if (edge.source !== connection.target) return false;
      const outputPort = to?.outputs.find((port) => port.name === edge.sourceHandle);
      return outputPort?.type !== input?.type;
    });
    const reachable = new Set<string>();
    const stack = [connection.target];
    while (stack.length) {
      const id = stack.pop()!;
      if (reachable.has(id)) continue;
      reachable.add(id);
      activeEdges
        .filter((edge) => edge.source === id)
        .forEach((edge) => stack.push(edge.target));
    }
    return (
      !!output &&
      !!input &&
      output.type === input.type &&
      (!replacementInputType || replacementInputType === output.type) &&
      !replacementOutputMismatch &&
      connection.source !== connection.target &&
      !reachable.has(connection.source) &&
      !activeEdges.some(
        (edge) =>
          edge.target === connection.target &&
          (to?.input_mode === "one" || edge.targetHandle === connection.targetHandle),
      )
    );
  };
  const connect = (connection: Connection) => {
    if (!compatible(connection)) {
      setFeedback(t("advancedFlows.invalidConnection"));
      return;
    }
    setEdges((old) => [
      ...old,
      {
        id: crypto.randomUUID(),
        source: connection.source,
        sourceHandle: connection.sourceHandle,
        target: connection.target,
        targetHandle: connection.targetHandle,
      },
    ]);
    if (connection.target && activeNodes.find((node) => node.id === connection.target)?.data.kind.type === "text.replace") {
      const attribute = connection.targetHandle === "candidate" ? "name" : connection.targetHandle === "text" ? "value" : "show_name";
      setNodes((old) => old.map((node) => node.id === connection.target
        ? { ...node, data: { ...node.data, config: { ...(node.data.config as Record<string, string>), attribute } } }
        : node));
    }
    setFeedback("");
  };
  const changeSelectedConfig = (key: string, value: string) => {
    setNodes((old) => old.map((node) => node.id === selected
      ? { ...node, data: { ...node.data, config: { ...(node.data.config as Record<string, string>), [key]: value } } }
      : node));
  };
  const addNode = (kind: Kind, position?: { x: number; y: number }) => {
    if (!current) {
      const draft: Flow = {
        id: "",
        name: t("advancedFlows.newName"),
        revision: 0,
        definition: { nodes: [], edges: [] },
        created_at: 0,
        updated_at: 0,
      };
      setCurrent(draft);
      setName(draft.name);
    }
    const id = crypto.randomUUID();
    setNodes((old) => [
      ...old.map((node) =>
        node.selected ? { ...node, selected: false } : node,
      ),
      {
        id,
        type: "flow",
        position: position || { x: 140 + old.length * 35, y: 360 + old.length * 20 },
        data: {
          kind,
          label: label(kind.type),
          config: kind.type === "text.replace" ? { trim: "true" } : {},
        },
      },
    ]);
    setEdges((old) =>
      old.map((edge) => (edge.selected ? { ...edge, selected: false } : edge)),
    );
    setSelected("");
    setSelectedEdge("");
  };
  const save = async () => {
    if (!current) return;
    setBusy(true);
    setFeedback("");
    try {
      const payload = {
        name,
        revision: current.revision,
        definition: definition(),
      };
      const saved: Flow = await api(
        current.id ? `/flows/${current.id}` : "/flows",
        current.id ? "PUT" : "POST",
        payload,
      );
      setCurrent(saved);
      await cache.invalidateQueries({ queryKey: ["flows", "list"] });
      setFeedback(t("advancedFlows.saved"));
    } catch (error) {
      setFeedback(String(error));
    } finally {
      setBusy(false);
    }
  };
  const deleteFlow = async () => {
    if (!current?.id) return;
    await api(`/flows/${current.id}`, "DELETE");
    setCurrent(null);
    setName("");
    setNodes([]);
    setEdges([]);
    setInspection(null);
    setRecorded(false);
    setSelected("");
    setSelectedEdge("");
    await Promise.all([
      cache.invalidateQueries({ queryKey: ["flows", "list"] }),
      cache.invalidateQueries({ queryKey: ["flows", "runs"] }),
    ]);
    setFeedback(t("advancedFlows.flowDeleted"));
  };
  const test = async () => {
    if (!current?.id || !sourceID || (sourceID === "custom" && !customReady)) return;
    setBusy(true);
    setFeedback("");
    try {
      const run: Run = await api(`/flows/${current.id}/replay`, "POST", {
        ...(sourceID === "custom"
          ? { event: customEvent }
          : { source_run_id: sourceID }),
        definition: hasChanges ? definition() : undefined,
      });
      setInspection(run);
      setVisibleSteps(0);
      setRecorded(false);
      await cache.invalidateQueries({ queryKey: ["flows", "runs"] });
      setFeedback(t("advancedFlows.testComplete"));
    } catch (error) {
      setFeedback(String(error));
    } finally {
      setBusy(false);
    }
  };
  const openRun = async (summary: RunSummary) => {
    if (!discardAllowed()) return;
    setBusy(true);
    try {
      const run: Run = await api(`/flows/runs/${summary.id}`);
      const flow = flows.data?.flows.find((item) => item.id === run.flow_id);
      if (flow) selectFlow(flow);
      setInspection(run);
      setVisibleSteps(run.steps.length);
      setRecorded(true);
      setSourceID(run.source_run_id || (run.event.kind === "show_available" ? "custom" : ""));
      if (run.event.kind === "show_available") {
        setCustomEvent({
          kind: "show_available",
          show_name: run.event.show_name,
          season: run.event.season,
          episode: run.event.episode,
        });
      }
      setSelected("");
    } catch (error) {
      setFeedback(String(error));
    } finally {
      setBusy(false);
    }
  };
  if (kinds.error || flows.error)
    return (
      <div className="page">
        <ErrorState
          error={
            kinds.error || flows.error || new Error("Flow data unavailable")
          }
          retry={() => {
            void kinds.refetch();
            void flows.refetch();
          }}
        />
      </div>
    );
  if (!kinds.data || !flows.data)
    return (
      <div className="page">
        <Busy />
      </div>
    );
  return (
    <div className="page advanced-flows-page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">{t("advancedFlows.experimental")}</span>
          <h1>
            {t("advancedFlows.title")}
            <span className="accent">.</span>
          </h1>
          <p>{t("advancedFlows.description")}</p>
        </div>
      </div>
      <TorrentTabs />
      <div className="advanced-layout">
        <div className="advanced-left-rail">
          <section className="panel advanced-flow-setup" aria-label={t("advancedFlows.flow")}>
            <button type="button" className="button advanced-new-flow" onClick={newFlow}>
              <Plus size={17} />
              {t("advancedFlows.newFlow")}
            </button>
            <label>
              <span>{t("advancedFlows.savedFlows")}</span>
              <select
                value={current?.id || ""}
                onChange={(event) => {
                  const flow = flows.data.flows.find((item) => item.id === event.target.value);
                  if (flow && discardAllowed()) selectFlow(flow);
                }}
              >
                <option value="">{t("advancedFlows.chooseFlow")}</option>
                {flows.data.flows.map((flow) => (
                  <option key={flow.id} value={flow.id}>{flow.name}</option>
                ))}
              </select>
            </label>
            <label>
              <span>{t("advancedFlows.name")}</span>
              <input
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={100}
                placeholder={t("advancedFlows.newName")}
              />
              <small>{t("advancedFlows.nameHelp")}</small>
            </label>
            <div className="advanced-flow-actions">
              <button
                type="button"
                className="button primary advanced-save-flow"
                disabled={!current || busy || (!!current.id && !hasChanges)}
                onClick={save}
              >
                {current?.id && !hasChanges ? <Check size={16} /> : <Save size={16} />}
                {current?.id && !hasChanges
                  ? t("advancedFlows.savedState")
                  : t("advancedFlows.saveChanges")}
              </button>
              {current?.id && (
                <button
                  type="button"
                  className="button danger advanced-delete-flow"
                  disabled={busy}
                  onClick={() => setConfirmFlowDelete(true)}
                >
                  <Trash2 size={15} />
                  {t("advancedFlows.deleteFlow")}
                </button>
              )}
            </div>
            <div className="advanced-event-setup">
              <label>
                <span>{t("advancedFlows.historicalEvent")}</span>
                <select
                  value={sourceID}
                  onChange={(event) => {
                    setSourceID(event.target.value);
                    setInspection(null);
                    setRecorded(false);
                  }}
                >
                  <option value="">{t("advancedFlows.chooseEvent")}</option>
                  {sourceID === "custom" && (
                    <option value="custom">{t("advancedFlows.customEvent")}</option>
                  )}
                  {history.data?.runs.map((run) => (
                    <option key={run.id} value={run.id}>
                      {run.show_name} S{String(run.season).padStart(2, "0")}E
                      {String(run.episode).padStart(2, "0")} {" / "}
                      {dateLabel(run.started_at)}
                    </option>
                  ))}
                </select>
              </label>
              {sourceID !== "custom" && (
                <button
                  type="button"
                  className="button advanced-create-event"
                  onClick={() => {
                    setSourceID("custom");
                    setInspection(null);
                    setRecorded(false);
                  }}
                >
                  <Plus size={15} />
                  {t("advancedFlows.createEvent")}
                </button>
              )}
              {sourceID === "custom" && (
                <div className="advanced-custom-event">
                  <label>
                    <span>{t("advancedFlows.showName")}</span>
                    <input
                      value={customEvent.show_name}
                      maxLength={200}
                      onChange={(event) => {
                        setCustomEvent((current) => ({ ...current, show_name: event.target.value }));
                        setInspection(null);
                        setRecorded(false);
                      }}
                    />
                  </label>
                  <label>
                    <span>{t("advancedFlows.season")}</span>
                    <input
                      type="number"
                      min={0}
                      max={100}
                      value={Number.isNaN(customEvent.season) ? "" : customEvent.season}
                      onChange={(event) => {
                        setCustomEvent((current) => ({ ...current, season: event.target.valueAsNumber }));
                        setInspection(null);
                        setRecorded(false);
                      }}
                    />
                  </label>
                  <label>
                    <span>{t("advancedFlows.episode")}</span>
                    <input
                      type="number"
                      min={1}
                      max={1000}
                      value={Number.isNaN(customEvent.episode) ? "" : customEvent.episode}
                      onChange={(event) => {
                        setCustomEvent((current) => ({ ...current, episode: event.target.valueAsNumber }));
                        setInspection(null);
                        setRecorded(false);
                      }}
                    />
                  </label>
                </div>
              )}
            </div>
            {inspection && (
              <button
                type="button"
                className="button advanced-recorded-toggle"
                onClick={() => setRecorded(!recorded)}
              >
                {recorded
                  ? t("advancedFlows.editCurrent")
                  : t("advancedFlows.viewRecorded")}
              </button>
            )}
            <button
              type="button"
              className="button primary advanced-test-flow"
              disabled={!current?.id || !sourceID || (sourceID === "custom" && !customReady) || busy}
              onClick={test}
            >
              <FlaskConical size={16} />
              {t("advancedFlows.testEvent")}
            </button>
            {!current?.id && <small className="advanced-test-help">{t("advancedFlows.saveBeforeTest")}</small>}
            {feedback && (
              <p role="status" className="advanced-feedback">{feedback}</p>
            )}
          </section>
          <aside className="panel advanced-sidebar">
            <div className="advanced-sidebar-heading">
              <div>
                <span className="eyebrow">{t("advancedFlows.buildLabel")}</span>
                <h2>{t("advancedFlows.nodesHeading")}</h2>
              </div>
              <span className="advanced-node-count">{nodes.length}</span>
            </div>
            <p className="advanced-sidebar-help">{t("advancedFlows.addHelp")}</p>
            {kinds.data.nodes.filter((kind) => !kind.deprecated).map((kind) => (
              <button
                type="button"
                key={kind.type}
                disabled={recorded}
                draggable={!recorded}
                onDragStart={(event) => {
                  event.dataTransfer.setData("text/plain", kind.type);
                  event.dataTransfer.setData(nodeDragType, kind.type);
                  event.dataTransfer.effectAllowed = "copy";
                }}
                onDragEnd={() => setCanvasDropActive(false)}
                onClick={() => addNode(kind)}
              >
                <GripVertical size={15} aria-hidden="true" />
                {label(kind.type)}
              </button>
            ))}
          </aside>
        </div>
        <div
          className={`advanced-canvas${canvasDropActive ? " is-drop-target" : ""}`}
          aria-label={t("advancedFlows.canvas")}
          onDragOver={(event) => {
            if (
              !recorded &&
              (event.dataTransfer.types.includes(nodeDragType) ||
                event.dataTransfer.types.includes("text/plain"))
            ) {
              event.preventDefault();
              event.dataTransfer.dropEffect = "copy";
              setCanvasDropActive(true);
            }
          }}
          onDragLeave={(event) => {
            if (
              !(event.relatedTarget instanceof Element) ||
              !event.currentTarget.contains(event.relatedTarget)
            )
              setCanvasDropActive(false);
          }}
          onDrop={(event) => {
            const draggedType =
              event.dataTransfer.getData(nodeDragType) ||
              event.dataTransfer.getData("text/plain");
            const kind = kindMap.get(draggedType);
            setCanvasDropActive(false);
            if (!kind || kind.deprecated || recorded || !screenToFlowPosition.current) return;
            event.preventDefault();
            addNode(
              kind,
              screenToFlowPosition.current({
                x: event.clientX,
                y: event.clientY,
              }),
            );
          }}
        >
          {!activeNodes.length && !recorded && (
            <div className="advanced-canvas-empty">
              <span className="advanced-canvas-empty-icon">
                <Workflow size={22} />
              </span>
              <strong>{t("advancedFlows.startWithTrigger")}</strong>
              <span>{t("advancedFlows.emptyCanvasHelp")}</span>
            </div>
          )}
          {canvasDropActive && (
            <div className="advanced-canvas-drop-hint">
              {t("advancedFlows.dropHere")}
            </div>
          )}
          <ReactFlow
            key={canvasKey}
            onInit={(instance) => {
              screenToFlowPosition.current = instance.screenToFlowPosition;
            }}
            nodes={activeNodes}
            edges={activeEdges}
            nodeTypes={nodeTypes}
            onNodesChange={recorded ? undefined : onNodesChange}
            onEdgesChange={recorded ? undefined : onEdgesChange}
            onConnect={recorded ? undefined : connect}
            isValidConnection={compatible}
            onConnectEnd={(_, state) => {
              if (!state.isValid && state.fromNode)
                setFeedback(t("advancedFlows.invalidConnection"));
            }}
            onNodeClick={(_, node) => {
              setSelected(node.id);
              setSelectedEdge("");
            }}
            onEdgeClick={(_, edge) => {
              setSelected("");
              setSelectedEdge(edge.id);
            }}
            nodesDraggable={!recorded}
            nodesConnectable={!recorded}
            elementsSelectable
            fitView={canvasKey > 0}
            deleteKeyCode={["Backspace", "Delete"]}
          >
            <Background />
            <Controls showInteractive={false} />
          </ReactFlow>
        </div>
        <aside className="panel advanced-inspector">
          <h2>{t("advancedFlows.inspector")}</h2>
          {inspection && (
            <div className="advanced-run-summary">
              <strong>
                {t("advancedFlows.testRun")}: {statusLabel(inspection.status)}
              </strong>
              <span>
                {inspection.event.show_name} S{inspection.event.season}E
                {inspection.event.episode}
              </span>
              <span>{inspection.duration_ms} ms</span>
              {!traceMatches && <span>{t("advancedFlows.graphChanged")}</span>}
            </div>
          )}
          {selected && (
            <>
              <div className="advanced-selection-heading">
                <div>
                  <span className="advanced-selection-label">
                    {t("advancedFlows.selectedNode")}
                  </span>
                  <h3>
                    {label(selectedNodeType)}
                  </h3>
                </div>
                {!recorded && (
                  <button
                    type="button"
                    className="button danger advanced-delete-node"
                    onClick={() => {
                      setNodes((old) =>
                        old.filter((node) => node.id !== selected),
                      );
                      setEdges((old) =>
                        old.filter(
                          (edge) =>
                            edge.source !== selected && edge.target !== selected,
                        ),
                      );
                      setSelected("");
                    }}
                  >
                    <Trash2 size={15} />
                    {t("advancedFlows.deleteNode")}
                  </button>
                )}
              </div>
              <p className="advanced-node-description">
                {t(
                  `advancedFlows.nodeDescriptions.${selectedNodeType.replace(".", "_")}`,
                )}
              </p>
              {!recorded &&
                nodes
                  .find((node) => node.id === selected)
                  ?.data.kind.config.map((key) => {
                    if (key === "attribute") {
                      const config = nodes.find((node) => node.id === selected)?.data.config as Record<string, string>;
                      const options = selectedInputType === "candidate"
                        ? ["name", "provider"]
                        : selectedInputType === "event" ? ["show_name"]
                        : selectedInputType === "text" ? ["value"] : [];
                      return (
                        <label key={key}>
                          {t("advancedFlows.config.attribute")}
                          <select
                            value={config?.attribute || ""}
                            onChange={(event) => changeSelectedConfig(key, event.target.value)}
                            disabled={!selectedInputType}
                          >
                            <option value="">{t(selectedInputType ? "advancedFlows.config.defaultField" : "advancedFlows.config.connectInput")}</option>
                            {options.map((option) => <option value={option} key={option}>{t(`advancedFlows.config.${option}`)}</option>)}
                          </select>
                        </label>
                      );
                    }
                    if (key === "trim") {
                      const config = nodes.find((node) => node.id === selected)?.data.config as Record<string, string>;
                      return (
                        <div className="advanced-trim-option" key={key}>
                          <label className="toggle-setting">
                            <input type="checkbox" role="switch" checked={config?.trim !== "false"}
                              onChange={(event) => changeSelectedConfig(key, event.target.checked ? "true" : "false")} />
                            <span>{t("advancedFlows.config.trim")}</span>
                          </label>
                          <small>{t("advancedFlows.config.trimHelp")}</small>
                        </div>
                      );
                    }
                    return (
                      <label key={key}>
                        {t(`advancedFlows.config.${key}`)}
                        <input
                          value={(nodes.find((node) => node.id === selected)?.data.config as Record<string, string>)?.[key] || ""}
                          onChange={(event) => changeSelectedConfig(key, event.target.value)}
                          maxLength={500}
                        />
                      </label>
                    );
                  })}
              {!recorded && selectedNodeType === "query.build" && (
                <p className="advanced-config-help">{t("advancedFlows.config.prefixHelp")}</p>
              )}
              {selectedPreview && (
                <div className="advanced-preview">
                  <strong>{t("advancedFlows.preview.title")}</strong>
                  <div className="advanced-preview-grid">
                    <div>
                      <h4>{t("advancedFlows.preview.input")}</h4>
                      {selectedPreview.input === undefined
                        ? <p>{t(`advancedFlows.preview.${selectedPreview.note || "needsInput"}`)}</p>
                        : <pre>{JSON.stringify(selectedPreview.input, null, 2)}</pre>}
                      {selectedPreview.previousInput && <small>{t("advancedFlows.preview.previousInput")}</small>}
                    </div>
                    <div>
                      <h4>{t("advancedFlows.preview.output")}</h4>
                      {selectedPreview.output === undefined
                        ? <p>{t(`advancedFlows.preview.${selectedPreview.note || "runTest"}`)}</p>
                        : <pre>{JSON.stringify(selectedPreview.output, null, 2)}</pre>}
                    </div>
                  </div>
                </div>
              )}
              {selectedStep && (
                <div className="advanced-step">
                  <strong>{statusLabel(selectedStep.status)}</strong>
                  <p>{selectedStep.summary}</p>
                  {rejected && (
                    <div>
                      <h4>{t("torrentRuns.filterRejected")}</h4>
                      {Object.entries(rejected.rejection_counts || {}).map(
                        ([code, count]) => (
                          <p key={code}>
                            {reasonLabel(code)}: {count}
                          </p>
                        ),
                      )}
                      {rejected.rejected?.map((candidate, index) => (
                        <details key={index}>
                          <summary>{candidate.name}</summary>
                          <ul>
                            {candidate.reasons.map((reason) => (
                              <li key={reason}>{reasonLabel(reason)}</li>
                            ))}
                          </ul>
                        </details>
                      ))}
                    </div>
                  )}
                  <details>
                    <summary>{t("advancedFlows.input")}</summary>
                    <pre>{JSON.stringify(selectedStep.input, null, 2)}</pre>
                  </details>
                  <details>
                    <summary>{t("advancedFlows.output")}</summary>
                    <pre>{JSON.stringify(selectedStep.output, null, 2)}</pre>
                  </details>
                  <span>{selectedStep.duration_ms} ms</span>
                </div>
              )}
            </>
          )}
          {selectedEdge && !recorded && (
            <div className="advanced-selection-heading advanced-connection-heading">
              <div>
                <span className="advanced-selection-label">
                  {t("advancedFlows.selectedConnection")}
                </span>
                <h3>
                  {label(
                    activeNodes.find(
                      (node) =>
                        node.id ===
                        activeEdges.find((edge) => edge.id === selectedEdge)
                          ?.source,
                    )?.data.kind.type || "",
                  )} →{" "}
                  {label(
                    activeNodes.find(
                      (node) =>
                        node.id ===
                        activeEdges.find((edge) => edge.id === selectedEdge)
                          ?.target,
                    )?.data.kind.type || "",
                  )}
                </h3>
              </div>
              <button
                type="button"
                className="button danger advanced-delete-node"
                onClick={() => {
                  setEdges((old) =>
                    old.filter((edge) => edge.id !== selectedEdge),
                  );
                  setSelectedEdge("");
                }}
              >
                <Trash2 size={15} />
                {t("advancedFlows.deleteConnection")}
              </button>
            </div>
          )}
          {original && (
            <div className="advanced-original">
              <h3>{t("advancedFlows.originalRun")}</h3>
              <p>
                {original.show_name} · {statusLabel(original.status)} ·{" "}
                {dateLabel(original.started_at)}
              </p>
              {original.decision_log.map((step, index) => (
                <p key={index}>
                  <strong>{step.stage}</strong> · {statusLabel(step.status)}
                  <br />
                  {step.summary}
                </p>
              ))}
            </div>
          )}
          {sourceID === "custom" && !inspection && (
            <div className="advanced-original">
              <h3>{t("advancedFlows.customEvent")}</h3>
              <pre>{JSON.stringify(customEvent, null, 2)}</pre>
            </div>
          )}
          {inspection && (
            <details className="advanced-original">
              <summary>{t("advancedFlows.eventData")}</summary>
              <pre>{JSON.stringify(inspection.event, null, 2)}</pre>
            </details>
          )}
          {!selected && !original && sourceID !== "custom" && <p>{t("advancedFlows.inspectHelp")}</p>}
        </aside>
      </div>
      <section className="panel advanced-history">
        <h2>{t("advancedFlows.runs")}</h2>
        {runs.data?.runs.length ? (
          runs.data.runs.map((run) => (
            <button
              type="button"
              key={run.id}
              disabled={busy}
              onClick={() => void openRun(run)}
            >
              {run.event.show_name} S{run.event.season}E{run.event.episode} ·{" "}
              {statusLabel(run.status)} · {dateLabel(run.started_at)}
            </button>
          ))
        ) : (
          <p>{t("advancedFlows.noRuns")}</p>
        )}
      </section>
      {confirmFlowDelete && current?.id && (
        <Confirm
          title={t("advancedFlows.deleteFlowTitle")}
          message={t("advancedFlows.deleteFlowMessage", { name: current.name })}
          onClose={() => setConfirmFlowDelete(false)}
          onConfirm={deleteFlow}
        />
      )}
    </div>
  );
}
