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
type Kind = { type: string; inputs: Port[]; outputs: Port[]; config: string[] };
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
  show_name: string;
  season: number;
  episode: number;
  status: string;
  started_at: number;
  query: string;
  decision_log: Array<{ stage: string; status: string; summary: string }>;
};
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
        <div className="advanced-port-row input" key={port.name}>
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
        <div className="advanced-port-row output" key={port.name}>
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
      connection.source !== connection.target &&
      !reachable.has(connection.source) &&
      !activeEdges.some(
        (edge) =>
          edge.target === connection.target &&
          edge.targetHandle === connection.targetHandle,
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
    setFeedback("");
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
        data: { kind, label: label(kind.type), config: {} },
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
    if (!current?.id || !sourceID) return;
    setBusy(true);
    setFeedback("");
    try {
      const run: Run = await api(`/flows/${current.id}/replay`, "POST", {
        source_run_id: sourceID,
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
      setSourceID(run.source_run_id || "");
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
      <div className="advanced-toolbar panel">
        <div className="advanced-toolbar-main">
          <div className="advanced-flow-picker">
            <label>
              <span>{t("advancedFlows.flow")}</span>
              <select
                aria-label={t("advancedFlows.flow")}
                value={current?.id || "new"}
                onChange={(event) => {
                  const flow = flows.data.flows.find(
                    (item) => item.id === event.target.value,
                  );
                  if (flow && discardAllowed()) selectFlow(flow);
                }}
              >
                <option value="new">
                  {current?.id
                    ? t("advancedFlows.chooseFlow")
                    : name || t("advancedFlows.newName")}
                </option>
                {flows.data.flows.map((flow) => (
                  <option key={flow.id} value={flow.id}>
                    {flow.name} · v{flow.revision}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              className="button advanced-new-flow"
              onClick={newFlow}
            >
              <Plus size={17} />
              {t("advancedFlows.newFlow")}
            </button>
          </div>
          <label className="advanced-flow-name">
            <span>{t("advancedFlows.name")}</span>
            <input
              aria-label={t("advancedFlows.name")}
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={100}
              placeholder={t("advancedFlows.newName")}
            />
            <small>{t("advancedFlows.nameHelp")}</small>
          </label>
          <div className="advanced-flow-actions">
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
            <button
              type="button"
              className="button primary advanced-save-flow"
              disabled={!current || busy || (!!current.id && !hasChanges)}
              onClick={save}
            >
              {current?.id && !hasChanges ? (
                <Check size={16} />
              ) : (
                <Save size={16} />
              )}
              {current?.id && !hasChanges
                ? t("advancedFlows.savedState")
                : t("advancedFlows.saveChanges")}
            </button>
          </div>
        </div>
        <div className="advanced-toolbar-replay">
          <label className="advanced-event-picker">
            <span>{t("advancedFlows.historicalEvent")}</span>
            <select
              aria-label={t("advancedFlows.historicalEvent")}
              value={sourceID}
              onChange={(event) => {
                setSourceID(event.target.value);
                setInspection(null);
              }}
            >
              <option value="">{t("advancedFlows.chooseEvent")}</option>
              {history.data?.runs.map((run) => (
                <option key={run.id} value={run.id}>
                  {run.show_name} S{String(run.season).padStart(2, "0")}E
                  {String(run.episode).padStart(2, "0")} ·{" "}
                  {dateLabel(run.started_at)}
                </option>
              ))}
            </select>
          </label>
          <div className="advanced-replay-actions">
            <button
              type="button"
              className="button advanced-test-flow"
              disabled={!current?.id || !sourceID || busy}
              onClick={test}
            >
              <FlaskConical size={16} />
              {t("advancedFlows.testEvent")}
            </button>
            {inspection && (
              <button
                type="button"
                className="button"
                onClick={() => setRecorded(!recorded)}
              >
                {recorded
                  ? t("advancedFlows.editCurrent")
                  : t("advancedFlows.viewRecorded")}
              </button>
            )}
          </div>
        </div>
      </div>
      {feedback && (
        <p role="status" className="advanced-feedback">
          {feedback}
        </p>
      )}
      <div className="advanced-layout">
        <aside className="panel advanced-sidebar">
          <div className="advanced-sidebar-heading">
            <div>
              <span className="eyebrow">{t("advancedFlows.buildLabel")}</span>
              <h2>{t("advancedFlows.nodesHeading")}</h2>
            </div>
            <span className="advanced-node-count">{nodes.length}</span>
          </div>
          <p className="advanced-sidebar-help">{t("advancedFlows.addHelp")}</p>
          {kinds.data.nodes.map((kind) => (
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
            if (!kind || recorded || !screenToFlowPosition.current) return;
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
              <span>
                {t("advancedFlows.revision")}:{" "}
                {inspection.flow_revision || t("advancedFlows.draft")}
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
                  ?.data.kind.config.map((key) => (
                    <label key={key}>
                      {t(`advancedFlows.config.${key}`)}
                      <input
                        value={
                          (
                            nodes.find((node) => node.id === selected)?.data
                              .config as Record<string, string>
                          )?.[key] || ""
                        }
                        onChange={(event) =>
                          setNodes((old) =>
                            old.map((node) =>
                              node.id === selected
                                ? {
                                    ...node,
                                    data: {
                                      ...node.data,
                                      config: {
                                        ...(node.data.config as Record<
                                          string,
                                          string
                                        >),
                                        [key]: event.target.value,
                                      },
                                    },
                                  }
                                : node,
                            ),
                          )
                        }
                        maxLength={500}
                      />
                    </label>
                  ))}
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
          {!selected && !original && <p>{t("advancedFlows.inspectHelp")}</p>}
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
