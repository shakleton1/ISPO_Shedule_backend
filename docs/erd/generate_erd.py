#!/usr/bin/env python3
"""Generate physical ERD fragments from schema_erd.json via Graphviz DOT."""

from __future__ import annotations

import html
import json
import re
import subprocess
import xml.etree.ElementTree as ET
from pathlib import Path


ROOT = Path(__file__).resolve().parent
SCHEMA_PATH = ROOT / "schema_erd.json"
SVG_NS = "http://www.w3.org/2000/svg"
EDGE_LEAD = 12.0
TABLE_GUARD = 4
CONNECTOR_GAP = 12.0
CONNECTOR_SPACING = 24.0

MANUAL_LAYOUTS = {
    "2_5a_schedule_replacements.svg": {
        "teachers": (45.0, -805.0),
        "groups": (45.0, -585.0),
        "subjects": (45.0, -190.0),
        "teacher_subjects": (260.0, -805.0),
        "course_assignments": (260.0, -650.0),
        "schedule_lessons": (260.0, -410.0),
        "schedule_overrides": (465.0, -720.0),
        "schedule_replacements": (465.0, -330.0),
    },
    "2_5b_rooms_constraints.svg": {
        "campuses": (35.0, -800.0),
        "location_types": (200.0, -800.0),
        "location_type_links": (375.0, -785.0),
        "room_requests": (525.0, -800.0),
        "locations": (35.0, -640.0),
        "teacher_location_preferences": (190.0, -625.0),
        "location_week_availability": (375.0, -625.0),
        "calendar_day_constraints": (525.0, -570.0),
        "room_assignments": (35.0, -435.0),
        "teacher_day_constraints": (200.0, -435.0),
        "schedule_day_events": (397.0, -405.0),
        "schedule_day_overlays": (525.0, -365.0),
    },
    "2_5c_curricula_users.svg": {
        "specialties": (35.0, -825.0),
        "study_activities": (205.0, -825.0),
        "students": (390.0, -840.0),
        "curricula": (35.0, -690.0),
        "study_calendar_weeks": (220.0, -620.0),
        "curriculum_items": (390.0, -690.0),
        "refresh_tokens": (540.0, -640.0),
        "device_tokens": (575.0, -760.0),
        "academic_calendars": (35.0, -505.0),
        "system_state": (205.0, -455.0),
        "audit_logs": (575.0, -450.0),
        "academic_calendar_weeks": (35.0, -380.0),
        "academic_calendar_day_overrides": (205.0, -380.0),
        "curriculum_item_allocations": (405.0, -430.0),
    }
}

MANUAL_VIEWBOX_WIDTHS = {
    "2_5a_schedule_replacements.svg": 690.0,
}

MANUAL_VIEWBOX_SIZES = {
    "2_5b_rooms_constraints.svg": (705.0, 625.0),
    "2_5c_curricula_users.svg": (720.0, 670.0),
}

MANUAL_CONNECTOR_SIDES = {
    "2_5a_schedule_replacements": {
        "teachers": "w",
        "groups": "w",
        "subjects": "w",
        "teacher_subjects": "w",
        "course_assignments": "e",
        "schedule_lessons": "e",
        "schedule_overrides": "e",
        "schedule_replacements": "e",
    },
    "2_5b_rooms_constraints": {
        "campuses": "w",
        "locations": "w",
        "room_requests": "e",
        "teacher_day_constraints": "e",
        "schedule_day_overlays": "e",
    },
    "2_5c_curricula_users": {
        "specialties": "w",
        "curricula": "w",
        "device_tokens": "e",
    }
}

MANUAL_INTERNAL_TAIL_SIDES = {
    "2_5a_schedule_replacements": {
        ("schedule_lessons", "schedule_overrides"): "w",
        ("schedule_overrides", "schedule_replacements"): "w",
    },
    "2_5b_rooms_constraints": {
        ("campuses", "locations"): "w",
        ("locations", "room_assignments"): "w",
    },
    "2_5c_curricula_users": {
        ("specialties", "curricula"): "w",
        ("curricula", "academic_calendars"): "w",
        ("academic_calendars", "academic_calendar_weeks"): "w",
        ("study_activities", "study_calendar_weeks"): "w",
    }
}

GRAPH_ATTRS = {
    "bgcolor": "white",
    "fontname": "Arial",
    "labelloc": "t",
    "labeljust": "l",
    "margin": "0.00",
    "nodesep": "0.14",
    "ranksep": "0.30",
    "pad": "0.02",
    "splines": "polyline",
    "outputorder": "edgesfirst",
    "overlap": "false",
}


def q(value: object) -> str:
    return json.dumps(str(value), ensure_ascii=False)


def html_text(value: object) -> str:
    return html.escape(str(value), quote=True)


def port_base(column: str) -> str:
    port = re.sub(r"[^A-Za-z0-9_]", "_", column)
    if not port or port[0].isdigit():
        port = f"_{port}"
    return port


def field_port(column: str, side: str) -> str:
    if side not in {"w", "e"}:
        raise ValueError(f"unsupported port side: {side}")
    return f"p_{port_base(column)}_{side}"


def endpoint(table: str, column: str | None = None, side: str = "e") -> str:
    if not column:
        return q(table)
    return f"{q(table)}:{q(field_port(column, side))}:{side}"


def connector_endpoint(connector: str, side: str) -> str:
    return f"{q(connector)}:{side}"


def opposite_side(side: str) -> str:
    return {"e": "w", "w": "e", "n": "s", "s": "n"}[side]


def connector_table_side(fragment_name: str, table_name: str, default: str) -> str:
    return MANUAL_CONNECTOR_SIDES.get(fragment_name, {}).get(table_name, default)


def safe_id_part(value: object) -> str:
    return re.sub(r"[^A-Za-z0-9_]+", "_", str(value)).strip("_") or "none"


def edge_id_for(tail_node: str, tail_column: str | None, head_node: str, head_column: str | None) -> str:
    return "edge__" + "__".join(
        safe_id_part(part)
        for part in (
            tail_node,
            tail_column or "node",
            head_node,
            head_column or "node",
        )
    )


def load_schema() -> dict:
    with SCHEMA_PATH.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def field_fk_names(table: dict) -> set[str]:
    result: set[str] = set()
    for fk in table.get("foreign_keys", []):
        result.update(fk["columns"])
    return result


def table_label(name: str, table: dict) -> str:
    fk_names = field_fk_names(table)

    rows = [
        f'<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="0" CELLPADDING="{TABLE_GUARD}"><TR><TD>',
        '<TABLE BORDER="1" CELLBORDER="1" CELLSPACING="0" CELLPADDING="1" COLOR="#000000">',
        (
            '<TR><TD BGCOLOR="#000000" COLSPAN="5">'
            f'<FONT FACE="Arial" COLOR="#ffffff" POINT-SIZE="11"><B>{html_text(name)}</B></FONT>'
            "</TD></TR>"
        ),
    ]

    for field in table["fields"]:
        markers: list[str] = []
        if field.get("pk"):
            markers.append("PK")
        if field["name"] in fk_names:
            markers.append("FK")
        marker = "/".join(markers) if markers else "&#160;"
        weight_open = "<B>" if field.get("pk") else ""
        weight_close = "</B>" if field.get("pk") else ""
        rows.append(
            "<TR>"
            f'<TD PORT="{field_port(field["name"], "w")}" WIDTH="2" BGCOLOR="#ffffff"><FONT FACE="Arial" POINT-SIZE="1" COLOR="#ffffff"> </FONT></TD>'
            f'<TD ALIGN="LEFT" BGCOLOR="#ffffff"><FONT FACE="Arial" POINT-SIZE="7" COLOR="#000000">{marker}</FONT></TD>'
            f'<TD ALIGN="LEFT">{weight_open}<FONT FACE="Arial" POINT-SIZE="8" COLOR="#000000">{html_text(field["name"])}</FONT>{weight_close}</TD>'
            f'<TD ALIGN="LEFT"><FONT FACE="Arial" POINT-SIZE="7" COLOR="#000000">{html_text(field["type"])}</FONT></TD>'
            f'<TD PORT="{field_port(field["name"], "e")}" WIDTH="2" BGCOLOR="#ffffff"><FONT FACE="Arial" POINT-SIZE="1" COLOR="#ffffff"> </FONT></TD>'
            "</TR>"
        )

    rows.append("</TABLE>")
    rows.append("</TD></TR></TABLE>")
    return "<\n" + "\n".join(rows) + "\n>"


def relationships(schema: dict) -> list[dict]:
    result: list[dict] = []
    for child_name, table in schema["tables"].items():
        for fk in table.get("foreign_keys", []):
            parent_name = fk["references"]["table"]
            result.append(
                {
                    "parent": parent_name,
                    "child": child_name,
                    "parent_columns": fk["references"]["columns"],
                    "child_columns": fk["columns"],
                }
            )
    return result


def table_fragments(schema: dict) -> dict[str, str]:
    result: dict[str, str] = {}
    for fragment_name, fragment in schema["fragments"].items():
        for table in fragment["tables"]:
            if table in result:
                raise ValueError(f"table {table} is declared in more than one fragment")
            result[table] = fragment_name
    return result


def relationship_key(relationship: dict) -> tuple[str, str, tuple[str, ...], tuple[str, ...]]:
    return (
        relationship["parent"],
        relationship["child"],
        tuple(relationship["parent_columns"]),
        tuple(relationship["child_columns"]),
    )


def connector_numbers(schema: dict, relationship_list: list[dict]) -> dict[tuple[str, str, tuple[str, ...], tuple[str, ...]], int]:
    by_table = table_fragments(schema)
    cross_relationships = [
        relationship
        for relationship in relationship_list
        if relationship["parent"] in by_table
        and relationship["child"] in by_table
        and by_table[relationship["parent"]] != by_table[relationship["child"]]
    ]
    remaining = {relationship_key(relationship): relationship for relationship in cross_relationships}

    ordered: list[dict] = []
    for item in schema.get("cross_connector_order", []):
        matches = [
            relationship
            for relationship in remaining.values()
            if relationship["parent"] == item["parent"] and relationship["child"] == item["child"]
        ]
        for relationship in sorted(matches, key=lambda rel: (rel["child_columns"], rel["parent_columns"])):
            ordered.append(relationship)
            remaining.pop(relationship_key(relationship), None)

    ordered.extend(
        sorted(
            remaining.values(),
            key=lambda rel: (rel["parent"], rel["child"], rel["child_columns"], rel["parent_columns"]),
        )
    )

    result: dict[tuple[str, str, tuple[str, ...], tuple[str, ...]], int] = {}
    for next_id, relationship in enumerate(ordered, start=1):
        result[relationship_key(relationship)] = next_id
    return result


def edge_attrs(
    *,
    constraint: bool = True,
    minlen: int | None = None,
    headclip: bool | None = False,
    tailclip: bool | None = False,
    edge_id: str | None = None,
) -> str:
    attrs = {
        "fontsize": "7",
        "color": "#000000",
        "penwidth": "1.0",
        "arrowsize": "0.45",
    }
    if not constraint:
        attrs["constraint"] = "false"
    if minlen is not None:
        attrs["minlen"] = str(minlen)
    if headclip is not None:
        attrs["headclip"] = "true" if headclip else "false"
    if tailclip is not None:
        attrs["tailclip"] = "true" if tailclip else "false"
    if edge_id is not None:
        attrs["id"] = edge_id
    return "[" + ", ".join(f"{key}={q(value)}" for key, value in attrs.items()) + "]"


def connector_node_name(connector_id: int) -> str:
    return f"connector_{connector_id:02d}"


def svg_points(path_data: str) -> list[tuple[float, float]]:
    values = [float(value) for value in re.findall(r"-?\d+(?:\.\d+)?", path_data)]
    if len(values) < 4:
        return []
    pairs = list(zip(values[0::2], values[1::2]))
    if "C" in path_data:
        return [pairs[0], *[pairs[index] for index in range(3, len(pairs), 3)]]
    return pairs


def is_same_point(left: tuple[float, float], right: tuple[float, float]) -> bool:
    return abs(left[0] - right[0]) < 0.05 and abs(left[1] - right[1]) < 0.05


def compact_points(points: list[tuple[float, float]]) -> list[tuple[float, float]]:
    result: list[tuple[float, float]] = []
    for point in points:
        if result and is_same_point(result[-1], point):
            continue
        result.append(point)

    changed = True
    while changed:
        changed = False
        compacted: list[tuple[float, float]] = []
        for point in result:
            compacted.append(point)
            while len(compacted) >= 3:
                a, b, c = compacted[-3:]
                same_x = abs(a[0] - b[0]) < 0.05 and abs(b[0] - c[0]) < 0.05
                same_y = abs(a[1] - b[1]) < 0.05 and abs(b[1] - c[1]) < 0.05
                b_between_x = min(a[0], c[0]) - 0.05 <= b[0] <= max(a[0], c[0]) + 0.05
                b_between_y = min(a[1], c[1]) - 0.05 <= b[1] <= max(a[1], c[1]) + 0.05
                if (same_x and b_between_y) or (same_y and b_between_x):
                    compacted.pop(-2)
                    changed = True
                else:
                    break
        result = compacted
    return result


def shifted_points_text(points_text: str, dx: float, dy: float) -> str:
    values = [float(value) for value in re.findall(r"-?\d+(?:\.\d+)?", points_text)]
    shifted = []
    for x, y in zip(values[0::2], values[1::2]):
        shifted.append(f"{x + dx:.2f},{y + dy:.2f}")
    return " ".join(shifted)


def shift_node_group(group: ET.Element, dx: float, dy: float) -> None:
    for polygon in group.findall(f"{{{SVG_NS}}}polygon"):
        polygon.set("points", shifted_points_text(polygon.attrib.get("points", ""), dx, dy))

    for ellipse in group.findall(f"{{{SVG_NS}}}ellipse"):
        ellipse.set("cx", f"{float(ellipse.attrib['cx']) + dx:.2f}")
        ellipse.set("cy", f"{float(ellipse.attrib['cy']) + dy:.2f}")

    for text in group.findall(f"{{{SVG_NS}}}text"):
        if "x" in text.attrib:
            text.set("x", f"{float(text.attrib['x']) + dx:.2f}")
        if "y" in text.attrib:
            text.set("y", f"{float(text.attrib['y']) + dy:.2f}")


def edge_node(endpoint_name: str) -> str:
    return endpoint_name.split(":", 1)[0]


def parse_edge_endpoint(endpoint_name: str) -> tuple[str, str | None]:
    parts = endpoint_name.split(":")
    if len(parts) > 1 and parts[-1] in {"n", "s", "e", "w"}:
        return ":".join(parts[:-1]), parts[-1]
    return endpoint_name, None


def edge_nodes(edge_title: str) -> set[str]:
    if "->" not in edge_title:
        return set()
    return {parse_edge_endpoint(part)[0] for part in edge_title.split("->", 1)}


def edge_tail_head_nodes(edge_title: str) -> tuple[str | None, str | None]:
    if "->" not in edge_title:
        return None, None
    tail, head = edge_title.split("->", 1)
    return parse_edge_endpoint(tail)[0], parse_edge_endpoint(head)[0]


def edge_endpoint_meta(node: str, column: str | None) -> dict[str, str | None]:
    return {"node": node, "column": column}


def point_inside_box(point: tuple[float, float], box: tuple[float, float, float, float]) -> bool:
    x, y = point
    min_x, min_y, max_x, max_y = box
    return min_x - 0.5 <= x <= max_x + 0.5 and min_y - 0.5 <= y <= max_y + 0.5


def adjust_lead_away_from_boxes(
    lead: tuple[float, float],
    side: str,
    owner_node: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> tuple[float, float]:
    x, y = lead
    margin = 3.0
    for node_name, box in boxes.items():
        if node_name == owner_node or node_name.startswith("connector_"):
            continue
        if not point_inside_box((x, y), box):
            continue

        min_x, min_y, max_x, max_y = box
        if side == "w":
            x = max_x + margin
        elif side == "e":
            x = min_x - margin
        elif side == "n":
            y = max_y + margin
        elif side == "s":
            y = min_y - margin
    return (x, y)


def lead_point(
    endpoint_name: str,
    point: tuple[float, float],
    is_tail: bool,
    boxes: dict[str, tuple[float, float, float, float]],
) -> tuple[float, float] | None:
    node_name, side = parse_edge_endpoint(endpoint_name)
    x, y = point

    if node_name.startswith("connector_"):
        if is_tail and side == "e":
            return (x + EDGE_LEAD, y)
        if not is_tail and side == "w":
            return (x - EDGE_LEAD, y)
        if is_tail and side == "w":
            return (x - EDGE_LEAD, y)
        if not is_tail and side == "e":
            return (x + EDGE_LEAD, y)
        if is_tail and side == "s":
            return (x, y + EDGE_LEAD)
        if not is_tail and side == "n":
            return (x, y - EDGE_LEAD)
        if is_tail and side == "n":
            return (x, y - EDGE_LEAD)
        if not is_tail and side == "s":
            return (x, y + EDGE_LEAD)
        return None

    if is_tail and side == "e":
        return adjust_lead_away_from_boxes((x + EDGE_LEAD, y), "e", node_name, boxes)
    if is_tail and side == "w":
        return adjust_lead_away_from_boxes((x - EDGE_LEAD, y), "w", node_name, boxes)
    if not is_tail and side == "w":
        return adjust_lead_away_from_boxes((x - EDGE_LEAD, y), "w", node_name, boxes)
    if not is_tail and side == "e":
        return adjust_lead_away_from_boxes((x + EDGE_LEAD, y), "e", node_name, boxes)
    return None


def apply_endpoint_leads(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> list[tuple[float, float]]:
    if len(points) < 2 or "->" not in edge_title:
        return points

    tail, head = edge_title.split("->", 1)
    start = points[0]
    end = points[-1]
    tail_lead = lead_point(tail, start, True, boxes)
    head_lead = lead_point(head, end, False, boxes)

    result = [start]
    if tail_lead and not is_same_point(tail_lead, start):
        result.append(tail_lead)

    result.extend(points[1:-1])

    if head_lead and not is_same_point(head_lead, end):
        result.append(head_lead)
    result.append(end)
    return compact_points(result)


def snap_endpoint_to_side(
    endpoint_name: str,
    point: tuple[float, float],
    boxes: dict[str, tuple[float, float, float, float]],
) -> tuple[float, float]:
    node_name, side = parse_edge_endpoint(endpoint_name)
    if side is None or node_name not in boxes:
        return point

    min_x, min_y, max_x, max_y = boxes[node_name]
    x, y = point
    if node_name.startswith("connector_"):
        center_x = (min_x + max_x) / 2
        center_y = (min_y + max_y) / 2
        if side == "w":
            return (min_x, center_y)
        if side == "e":
            return (max_x, center_y)
        if side == "n":
            return (center_x, min_y)
        if side == "s":
            return (center_x, max_y)

    if side == "w":
        return (min_x, y)
    if side == "e":
        return (max_x, y)
    if side == "n":
        return (x, min_y)
    if side == "s":
        return (x, max_y)
    return point


def snap_edge_endpoints(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
    node_shifts: dict[str, tuple[float, float]] | None = None,
    meta: dict[str, dict[str, str | None]] | None = None,
    field_centers: dict[str, dict[str, float]] | None = None,
) -> list[tuple[float, float]]:
    if len(points) < 2 or "->" not in edge_title:
        return points

    tail, head = edge_title.split("->", 1)
    snapped = list(points)
    if node_shifts:
        tail_node, _ = parse_edge_endpoint(tail)
        head_node, _ = parse_edge_endpoint(head)
        if tail_node in node_shifts:
            dx, dy = node_shifts[tail_node]
            snapped[0] = (snapped[0][0] + dx, snapped[0][1] + dy)
        if head_node in node_shifts:
            dx, dy = node_shifts[head_node]
            snapped[-1] = (snapped[-1][0] + dx, snapped[-1][1] + dy)
    if meta and field_centers:
        tail_meta = meta["tail"]
        head_meta = meta["head"]
        tail_node = tail_meta["node"]
        tail_column = tail_meta["column"]
        head_node = head_meta["node"]
        head_column = head_meta["column"]
        if tail_node and tail_column and tail_column in field_centers.get(tail_node, {}):
            snapped[0] = (snapped[0][0], field_centers[tail_node][tail_column])
        if head_node and head_column and head_column in field_centers.get(head_node, {}):
            snapped[-1] = (snapped[-1][0], field_centers[head_node][head_column])
    snapped[0] = snap_endpoint_to_side(tail, snapped[0], boxes)
    snapped[-1] = snap_endpoint_to_side(head, snapped[-1], boxes)
    return snapped


def trim_boundary_points_before_head(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> list[tuple[float, float]]:
    if len(points) < 3 or "->" not in edge_title:
        return points

    _, head = edge_title.split("->", 1)
    node_name, side = parse_edge_endpoint(head)
    if side is None or node_name not in boxes:
        return points

    min_x, min_y, max_x, max_y = boxes[node_name]
    boundary = {
        "w": min_x,
        "e": max_x,
        "n": min_y,
        "s": max_y,
    }[side]

    trimmed = list(points)
    while len(trimmed) >= 3:
        previous = trimmed[-2]
        coordinate = previous[0] if side in {"w", "e"} else previous[1]
        if abs(coordinate - boundary) > 2.0:
            break
        trimmed.pop(-2)
    return trimmed


def segment_crosses_box(
    start: tuple[float, float],
    end: tuple[float, float],
    box: tuple[float, float, float, float],
) -> bool:
    min_x, min_y, max_x, max_y = box
    min_x += 1.0
    min_y += 1.0
    max_x -= 1.0
    max_y -= 1.0
    if min_x >= max_x or min_y >= max_y:
        return False
    if (
        min(start[0], end[0]) > max_x
        or max(start[0], end[0]) < min_x
        or min(start[1], end[1]) > max_y
        or max(start[1], end[1]) < min_y
    ):
        return False
    if abs(start[0] - end[0]) < 0.05:
        return min_x <= start[0] <= max_x and max(min(start[1], end[1]), min_y) <= min(max(start[1], end[1]), max_y)
    if abs(start[1] - end[1]) < 0.05:
        return min_y <= start[1] <= max_y and max(min(start[0], end[0]), min_x) <= min(max(start[0], end[0]), max_x)
    return True


def path_crossing_score(
    points: list[tuple[float, float]],
    boxes: dict[str, tuple[float, float, float, float]],
    ignored_nodes: set[str],
) -> int:
    score = 0
    for start, end in zip(points, points[1:]):
        for node_name, box in boxes.items():
            if node_name in ignored_nodes:
                continue
            if segment_crosses_box(start, end, box):
                score += 1
    return score


def routed_path_crossing_score(
    points: list[tuple[float, float]],
    boxes: dict[str, tuple[float, float, float, float]],
    tail_node: str,
    head_node: str,
) -> int:
    score = 0
    last_segment = len(points) - 2
    for index, (start, end) in enumerate(zip(points, points[1:])):
        for node_name, box in boxes.items():
            if node_name == tail_node and index == 0:
                continue
            if node_name == head_node and index == last_segment:
                continue
            if segment_crosses_box(start, end, box):
                score += 1
    return score


def manhattan_points(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> list[tuple[float, float]]:
    if len(points) < 2:
        return points

    tail_node, head_node = edge_tail_head_nodes(edge_title)
    ignored_nodes = set()
    if tail_node and tail_node != head_node:
        ignored_nodes.add(tail_node)
    result = [points[0]]
    for index, point in enumerate(points[1:], start=1):
        previous = result[-1]
        if abs(previous[0] - point[0]) < 0.05 or abs(previous[1] - point[1]) < 0.05:
            result.append(point)
            continue

        is_first = index == 1
        is_last = index == len(points) - 1
        previous_segment = None
        if len(result) >= 2:
            before = result[-2]
            previous_segment = "h" if abs(before[1] - previous[1]) < 0.05 else "v"

        horizontal_first = (point[0], previous[1])
        vertical_first = (previous[0], point[1])

        candidates = [
            [previous, horizontal_first, point],
            [previous, vertical_first, point],
        ]
        scores = [path_crossing_score(candidate, boxes, ignored_nodes) for candidate in candidates]

        if is_first:
            preferred_index = 0
        elif is_last:
            preferred_index = 1
        elif previous_segment == "h":
            preferred_index = 1
        else:
            preferred_index = 0

        other_index = 1 - preferred_index
        corner = candidates[preferred_index][1] if scores[preferred_index] <= scores[other_index] else candidates[other_index][1]

        if not is_same_point(previous, corner):
            result.append(corner)
        result.append(point)

    if len(result) == 2 and result[0][0] != result[1][0] and result[0][1] != result[1][1]:
        start, end = result
        middle_x = (start[0] + end[0]) / 2
        result = [start, (middle_x, start[1]), (middle_x, end[1]), end]

    return compact_points(result)


def path_data(points: list[tuple[float, float]]) -> str:
    if not points:
        return ""
    commands = [f"M{points[0][0]:.2f},{points[0][1]:.2f}"]
    commands.extend(f"L{x:.2f},{y:.2f}" for x, y in points[1:])
    return "".join(commands)


def arrow_polygon(points: list[tuple[float, float]]) -> str | None:
    if len(points) < 2:
        return None

    end = points[-1]
    previous = None
    for candidate in reversed(points[:-1]):
        if not is_same_point(candidate, end):
            previous = candidate
            break
    if previous is None:
        return None

    dx = end[0] - previous[0]
    dy = end[1] - previous[1]
    length = (dx * dx + dy * dy) ** 0.5
    if length < 0.05:
        return None

    ux = dx / length
    uy = dy / length
    px = -uy
    py = ux
    tip_len = 4.5
    half_width = 2.0

    tip = end
    base_center = (end[0] - ux * tip_len, end[1] - uy * tip_len)
    base_left = (base_center[0] + px * half_width, base_center[1] + py * half_width)
    base_right = (base_center[0] - px * half_width, base_center[1] - py * half_width)
    return " ".join(f"{x:.2f},{y:.2f}" for x, y in (base_left, tip, base_right))


def connector_sort_key(connector: str) -> int:
    match = re.search(r"(\d+)$", connector)
    return int(match.group(1)) if match else 0


def spread_positions(
    items: list[tuple[str, float]],
    lower_bound: float | None = None,
    upper_bound: float | None = None,
) -> dict[str, float]:
    if not items:
        return {}

    sorted_items = sorted(items, key=lambda item: (item[1], connector_sort_key(item[0])))
    values = [base_y for _, base_y in sorted_items]
    for index in range(1, len(values)):
        min_allowed = values[index - 1] + CONNECTOR_SPACING
        if values[index] < min_allowed:
            values[index] = min_allowed
    for index in range(len(values) - 2, -1, -1):
        max_allowed = values[index + 1] - CONNECTOR_SPACING
        if values[index] > max_allowed:
            values[index] = max_allowed

    original_center = sum(base_y for _, base_y in sorted_items) / len(sorted_items)
    adjusted_center = sum(values) / len(values)
    shift = original_center - adjusted_center
    values = [y + shift for y in values]

    if lower_bound is not None and upper_bound is not None and values:
        stack_height = values[-1] - values[0]
        available_height = upper_bound - lower_bound
        if stack_height <= available_height:
            if values[0] < lower_bound:
                values = [y + (lower_bound - values[0]) for y in values]
            if values[-1] > upper_bound:
                values = [y + (upper_bound - values[-1]) for y in values]

    return {connector: y for (connector, _), y in zip(sorted_items, values)}


def move_connector_node(root: ET.Element, connector: str, x: float, y: float) -> tuple[float, float, float, float] | None:
    for group in root.findall(f".//{{{SVG_NS}}}g"):
        if group.attrib.get("class") != "node":
            continue
        title = group.find(f"{{{SVG_NS}}}title")
        if title is None or title.text != connector:
            continue

        ellipse = group.find(f"{{{SVG_NS}}}ellipse")
        if ellipse is None:
            return None

        old_x = float(ellipse.attrib["cx"])
        old_y = float(ellipse.attrib["cy"])
        dx = x - old_x
        dy = y - old_y
        rx = float(ellipse.attrib["rx"])
        ry = float(ellipse.attrib["ry"])
        ellipse.set("cx", f"{x:.2f}")
        ellipse.set("cy", f"{y:.2f}")

        for text in group.findall(f"{{{SVG_NS}}}text"):
            text.set("x", f"{float(text.attrib['x']) + dx:.2f}")
            text.set("y", f"{float(text.attrib['y']) + dy:.2f}")

        return (x - rx, y - ry, x + rx, y + ry)
    return None


def field_row_centers(root: ET.Element) -> dict[str, dict[str, float]]:
    result: dict[str, dict[str, float]] = {}
    for group in root.findall(f".//{{{SVG_NS}}}g"):
        if group.attrib.get("class") != "node":
            continue
        title = group.find(f"{{{SVG_NS}}}title")
        if title is None or not title.text or title.text.startswith("connector_"):
            continue

        cells: list[tuple[float, float, float, float, float]] = []
        for polygon in group.findall(f"{{{SVG_NS}}}polygon"):
            if polygon.attrib.get("stroke") == "none":
                continue
            values = [float(value) for value in re.findall(r"-?\d+(?:\.\d+)?", polygon.attrib.get("points", ""))]
            if len(values) < 8:
                continue
            xs = values[0::2]
            ys = values[1::2]
            min_x, max_x = min(xs), max(xs)
            min_y, max_y = min(ys), max(ys)
            area = (max_x - min_x) * (max_y - min_y)
            cells.append((min_x, min_y, max_x, max_y, area))

        table_fields: dict[str, float] = {}
        for text in group.findall(f"{{{SVG_NS}}}text"):
            if not text.text or "x" not in text.attrib or "y" not in text.attrib:
                continue
            x = float(text.attrib["x"])
            y = float(text.attrib["y"])
            containing = [
                cell
                for cell in cells
                if cell[0] - 0.1 <= x <= cell[2] + 0.1 and cell[1] - 0.1 <= y <= cell[3] + 0.1
            ]
            if not containing:
                continue
            min_x, min_y, max_x, max_y, _ = min(containing, key=lambda cell: cell[4])
            # Field-name cells are neither the marker column nor the type column.
            if max_x - min_x < 18.0:
                continue
            table_fields[text.text] = (min_y + max_y) / 2
        result[title.text] = table_fields

    return result


def apply_manual_layout(
    root: ET.Element,
    svg_name: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> dict[str, tuple[float, float]]:
    layout = MANUAL_LAYOUTS.get(svg_name)
    if not layout:
        return {}

    shifts: dict[str, tuple[float, float]] = {}
    for group in root.findall(f".//{{{SVG_NS}}}g"):
        if group.attrib.get("class") != "node":
            continue
        title = group.find(f"{{{SVG_NS}}}title")
        if title is None or not title.text or title.text not in layout or title.text not in boxes:
            continue

        target_x, target_y = layout[title.text]
        min_x, min_y, max_x, max_y = boxes[title.text]
        dx = target_x - min_x
        dy = target_y - min_y
        shift_node_group(group, dx, dy)
        boxes[title.text] = (min_x + dx, min_y + dy, max_x + dx, max_y + dy)
        shifts[title.text] = (dx, dy)

    return shifts


def apply_manual_viewbox(root: ET.Element, svg_name: str) -> None:
    size = MANUAL_VIEWBOX_SIZES.get(svg_name)
    width = size[0] if size else MANUAL_VIEWBOX_WIDTHS.get(svg_name)
    height = size[1] if size else None
    if width is None:
        return

    view_box = root.attrib.get("viewBox", "")
    values = [float(value) for value in re.findall(r"-?\d+(?:\.\d+)?", view_box)]
    if len(values) != 4:
        return

    values[2] = width
    if height is not None:
        values[3] = height
    root.set("viewBox", " ".join(f"{value:.2f}" for value in values))
    root.set("width", f"{width:.0f}pt")
    if height is not None:
        root.set("height", f"{height:.0f}pt")

    graph = root.find(f".//{{{SVG_NS}}}g[@class='graph']")
    if graph is not None:
        for polygon in graph.findall(f"{{{SVG_NS}}}polygon"):
            if polygon.attrib.get("fill") != "white":
                continue
            points = polygon.attrib.get("points", "")
            point_values = [float(value) for value in re.findall(r"-?\d+(?:\.\d+)?", points)]
            if len(point_values) < 8:
                continue
            max_x = max(point_values[0::2])
            min_y = min(point_values[1::2])
            dx = width - max_x
            shifted = []
            for x, y in zip(point_values[0::2], point_values[1::2]):
                shifted_x = x + dx if abs(x - max_x) < 0.05 else x
                shifted_y = -height if height is not None and abs(y - min_y) < 0.05 else y
                shifted.append(f"{shifted_x:.2f},{shifted_y:.2f}")
            polygon.set("points", " ".join(shifted))
            break


def place_connectors(
    root: ET.Element,
    boxes: dict[str, tuple[float, float, float, float]],
    node_shifts: dict[str, tuple[float, float]],
    edge_meta: dict[str, dict[str, dict[str, str | None]]],
    field_centers: dict[str, dict[str, float]],
) -> None:
    placements: dict[tuple[str, str], list[tuple[str, float]]] = {}

    for group in root.findall(f".//{{{SVG_NS}}}g"):
        if group.attrib.get("class") != "edge":
            continue

        title = group.find(f"{{{SVG_NS}}}title")
        path = group.find(f"{{{SVG_NS}}}path")
        if title is None or path is None or not title.text or "->" not in title.text:
            continue

        meta = edge_meta.get(group.attrib.get("id", ""))
        tail, head = title.text.split("->", 1)
        tail_node, tail_side = parse_edge_endpoint(tail)
        head_node, head_side = parse_edge_endpoint(head)
        if tail_node.startswith("connector_") == head_node.startswith("connector_"):
            continue

        points = svg_points(path.attrib.get("d", ""))
        if len(points) < 2:
            continue

        if tail_node.startswith("connector_"):
            connector = tail_node
            table = head_node
            table_side = head_side
            table_point = points[-1]
        else:
            connector = head_node
            table = tail_node
            table_side = tail_side
            table_point = points[0]

        table_column = None
        if meta:
            if meta["tail"]["node"] == table:
                table_column = meta["tail"]["column"]
            elif meta["head"]["node"] == table:
                table_column = meta["head"]["column"]

        if table_column and table_column in field_centers.get(table, {}):
            table_point = (table_point[0], field_centers[table][table_column])
        elif table in node_shifts:
            dx, dy = node_shifts[table]
            table_point = (table_point[0] + dx, table_point[1] + dy)

        if table_side not in {"e", "w"} or table not in boxes or connector not in boxes:
            continue

        placements.setdefault((table, table_side), []).append((connector, table_point[1]))

    for (table, table_side), connector_items in placements.items():
        min_x, min_y, max_x, max_y = boxes[table]
        y_by_connector = spread_positions(connector_items, min_y + 10.0, max_y - 10.0)
        for connector, y in y_by_connector.items():
            connector_box = boxes[connector]
            radius = (connector_box[2] - connector_box[0]) / 2
            x = max_x + CONNECTOR_GAP + radius if table_side == "e" else min_x - CONNECTOR_GAP - radius
            moved_box = move_connector_node(root, connector, x, y)
            if moved_box:
                boxes[connector] = moved_box


def self_edge_points(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> list[tuple[float, float]] | None:
    if len(points) < 2 or "->" not in edge_title:
        return None

    tail, head = edge_title.split("->", 1)
    tail_node, tail_side = parse_edge_endpoint(tail)
    head_node, head_side = parse_edge_endpoint(head)
    if tail_node != head_node or tail_node not in boxes or tail_side not in {"e", "w"} or head_side not in {"e", "w"}:
        return None

    min_x, min_y, max_x, max_y = boxes[tail_node]
    start = points[0]
    end = points[-1]
    left_x = min_x - EDGE_LEAD
    right_x = max_x + EDGE_LEAD

    if tail_side == head_side == "e":
        return [start, (right_x, start[1]), (right_x, end[1]), end]
    if tail_side == head_side == "w":
        return [start, (left_x, start[1]), (left_x, end[1]), end]

    outside_y_values = [max_y + EDGE_LEAD, min_y - EDGE_LEAD]

    candidates: list[list[tuple[float, float]]] = []
    for outside_y in outside_y_values:
        candidates.append([start, (right_x, start[1]), (right_x, outside_y), (left_x, outside_y), (left_x, end[1]), end])

    ignored_nodes = {tail_node}
    return min(candidates, key=lambda candidate: path_crossing_score(candidate, boxes, ignored_nodes))


def connector_edge_points(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> list[tuple[float, float]] | None:
    if len(points) < 2 or "->" not in edge_title:
        return None

    tail, head = edge_title.split("->", 1)
    tail_node, tail_side = parse_edge_endpoint(tail)
    head_node, head_side = parse_edge_endpoint(head)
    has_connector = tail_node.startswith("connector_") or head_node.startswith("connector_")
    if not has_connector or tail_node.startswith("connector_") == head_node.startswith("connector_"):
        return None
    if tail_side not in {"e", "w"} or head_side not in {"e", "w"}:
        return None

    start = points[0]
    end = points[-1]

    lead = EDGE_LEAD
    if tail_side == "e" and head_side == "w" and end[0] > start[0]:
        lead = min(EDGE_LEAD, max(1.5, (end[0] - start[0]) / 3))
    elif tail_side == "w" and head_side == "e" and start[0] > end[0]:
        lead = min(EDGE_LEAD, max(1.5, (start[0] - end[0]) / 3))

    tail_dx = lead if tail_side == "e" else -lead
    head_dx = -lead if head_side == "w" else lead
    tail_lead = (start[0] + tail_dx, start[1])
    head_lead = (end[0] + head_dx, end[1])

    candidates = [
        [start, tail_lead, (tail_lead[0], head_lead[1]), head_lead, end],
        [start, tail_lead, (head_lead[0], tail_lead[1]), head_lead, end],
    ]
    ignored_nodes = {tail_node, head_node}
    return compact_points(
        min(
            candidates,
            key=lambda candidate: (
                path_crossing_score(candidate, boxes, ignored_nodes),
                sum(abs(a[0] - b[0]) + abs(a[1] - b[1]) for a, b in zip(candidate, candidate[1:])),
            ),
        )
    )


def table_edge_points(
    points: list[tuple[float, float]],
    edge_title: str,
    boxes: dict[str, tuple[float, float, float, float]],
) -> list[tuple[float, float]] | None:
    if len(points) < 2 or "->" not in edge_title:
        return None

    tail, head = edge_title.split("->", 1)
    tail_node, tail_side = parse_edge_endpoint(tail)
    head_node, head_side = parse_edge_endpoint(head)
    if tail_node.startswith("connector_") or head_node.startswith("connector_") or tail_node == head_node:
        return None
    if tail_side not in {"e", "w"} or head_side not in {"e", "w"}:
        return None

    start = points[0]
    end = points[-1]
    tail_lead = lead_point(tail, start, True, boxes)
    head_lead = lead_point(head, end, False, boxes)
    if tail_lead is None or head_lead is None:
        return None

    table_boxes = [box for node_name, box in boxes.items() if not node_name.startswith("connector_")]
    all_min_x = min(box[0] for box in table_boxes)
    all_max_x = max(box[2] for box in table_boxes)
    all_min_y = min(box[1] for box in table_boxes)
    all_max_y = max(box[3] for box in table_boxes)
    mid_x = (tail_lead[0] + head_lead[0]) / 2
    mid_y = (tail_lead[1] + head_lead[1]) / 2
    left_lane = all_min_x - 18.0
    right_lane = all_max_x + 18.0
    top_lane = all_min_y - 18.0
    bottom_lane = all_max_y + 18.0

    candidates = [
        [start, tail_lead, (tail_lead[0], head_lead[1]), head_lead, end],
        [start, tail_lead, (head_lead[0], tail_lead[1]), head_lead, end],
        [start, tail_lead, (mid_x, tail_lead[1]), (mid_x, head_lead[1]), head_lead, end],
        [start, tail_lead, (tail_lead[0], mid_y), (head_lead[0], mid_y), head_lead, end],
        [start, tail_lead, (tail_lead[0], top_lane), (head_lead[0], top_lane), head_lead, end],
        [start, tail_lead, (tail_lead[0], bottom_lane), (head_lead[0], bottom_lane), head_lead, end],
        [start, tail_lead, (left_lane, tail_lead[1]), (left_lane, head_lead[1]), head_lead, end],
        [start, tail_lead, (right_lane, tail_lead[1]), (right_lane, head_lead[1]), head_lead, end],
    ]
    def route_cost(candidate: list[tuple[float, float]]) -> tuple[int, float, int]:
        compacted = compact_points(candidate)
        length = sum(abs(a[0] - b[0]) + abs(a[1] - b[1]) for a, b in zip(compacted, compacted[1:]))
        return (routed_path_crossing_score(compacted, boxes, tail_node, head_node), length, len(compacted))

    return compact_points(min(candidates, key=route_cost))


def orthogonalize_svg(svg_path: Path, edge_meta: dict[str, dict[str, dict[str, str | None]]]) -> None:
    ET.register_namespace("", SVG_NS)
    tree = ET.parse(svg_path)
    root = tree.getroot()

    boxes: dict[str, tuple[float, float, float, float]] = {}
    for group in root.findall(f".//{{{SVG_NS}}}g"):
        if group.attrib.get("class") != "node":
            continue
        title = group.find(f"{{{SVG_NS}}}title")
        if title is None or not title.text:
            continue
        values: list[float] = []
        for polygon in group.findall(f"{{{SVG_NS}}}polygon"):
            if polygon.attrib.get("stroke") == "none":
                continue
            values.extend([float(value) for value in re.findall(r"-?\d+(?:\.\d+)?", polygon.attrib.get("points", ""))])
        for ellipse in group.findall(f"{{{SVG_NS}}}ellipse"):
            cx = float(ellipse.attrib["cx"])
            cy = float(ellipse.attrib["cy"])
            rx = float(ellipse.attrib["rx"])
            ry = float(ellipse.attrib["ry"])
            values.extend([cx - rx, cy - ry, cx + rx, cy + ry])
        if values:
            xs = values[0::2]
            ys = values[1::2]
            boxes[title.text] = (min(xs), min(ys), max(xs), max(ys))

    manual_layout = svg_path.name in MANUAL_LAYOUTS
    node_shifts = apply_manual_layout(root, svg_path.name, boxes)
    field_centers = field_row_centers(root)
    place_connectors(root, boxes, node_shifts, edge_meta, field_centers)

    for group in root.findall(f".//{{{SVG_NS}}}g"):
        if group.attrib.get("class") != "edge":
            continue
        edge_id = group.attrib.get("id", "")
        if edge_id not in edge_meta:
            continue

        title = group.find(f"{{{SVG_NS}}}title")
        edge_title = title.text if title is not None else ""
        path = group.find(f"{{{SVG_NS}}}path")
        if path is None:
            continue

        snapped_points = snap_edge_endpoints(
            svg_points(path.attrib.get("d", "")),
            edge_title,
            boxes,
            node_shifts,
            edge_meta[edge_id],
            field_centers,
        )
        if manual_layout:
            points = connector_edge_points(snapped_points, edge_title, boxes)
            if points is None:
                points = self_edge_points(snapped_points, edge_title, boxes)
            if points is None:
                points = table_edge_points(snapped_points, edge_title, boxes)
            if points is None:
                points = snapped_points
        else:
            points = connector_edge_points(snapped_points, edge_title, boxes)
            if points is None:
                points = self_edge_points(snapped_points, edge_title, boxes)
            if points is None:
                snapped_points = trim_boundary_points_before_head(snapped_points, edge_title, boxes)
                raw_points = apply_endpoint_leads(snapped_points, edge_title, boxes)
                points = manhattan_points(raw_points, edge_title, boxes)
        if len(points) < 2:
            continue

        path.set("d", path_data(points))

        polygon_points = arrow_polygon(points)
        if polygon_points is None:
            continue
        for polygon in group.findall(f"{{{SVG_NS}}}polygon"):
            if polygon.attrib.get("fill") == "#000000":
                polygon.set("points", polygon_points)
                break

    apply_manual_viewbox(root, svg_path.name)
    tree.write(svg_path, encoding="utf-8", xml_declaration=False)


def append_rank_groups(
    lines: list[str],
    ranks: list[list[str]],
    left_connectors: dict[str, list[str]],
    right_connectors: dict[str, list[str]],
) -> None:
    expanded_ranks: list[list[str]] = []
    for rank in ranks:
        expanded_rank: list[str] = []
        for table in rank:
            expanded_rank.extend(sorted(set(left_connectors.get(table, []))))
            expanded_rank.append(table)
            expanded_rank.extend(sorted(set(right_connectors.get(table, []))))
        expanded_ranks.append(expanded_rank)

        lines.append("  { rank=same; " + " ".join(f"{q(node)};" for node in expanded_rank) + " }")
        for left, right in zip(expanded_rank, expanded_rank[1:]):
            lines.append(
                f"  {q(left)} -> {q(right)} "
                '[style="invis", weight="200"];'
            )

    for left_rank, right_rank in zip(expanded_ranks, expanded_ranks[1:]):
        if left_rank and right_rank:
            lines.append(
                f"  {q(left_rank[0])} -> {q(right_rank[0])} "
                '[style="invis", weight="120", minlen="1"];'
            )


def render_fragment(schema: dict, fragment_name: str, fragment: dict) -> tuple[str, dict[str, dict[str, dict[str, str | None]]]]:
    tables = schema["tables"]
    relationship_list = relationships(schema)
    table_to_fragment = table_fragments(schema)
    connector_ids = connector_numbers(schema, relationship_list)
    fragment_tables = set(fragment["tables"])

    missing = sorted(fragment_tables - set(tables))
    if missing:
        raise ValueError(f"{fragment_name}: missing table definitions: {', '.join(missing)}")

    graph_attrs = dict(GRAPH_ATTRS)
    graph_attrs["rankdir"] = fragment.get("rankdir", "LR")
    graph_attrs["label"] = fragment["title"]

    lines = ["digraph ERD {"]
    lines.append(
        "  graph ["
        + ", ".join(f"{key}={q(value)}" for key, value in graph_attrs.items())
        + "];"
    )
    lines.append('  node [shape="plain", fontname="Arial"];')
    lines.append(
        '  edge [fontname="Arial", fontsize="7", color="#000000", '
        'fontcolor="#000000", arrowsize="0.45"];'
    )
    lines.append("")

    for table_name in fragment["tables"]:
        lines.append(f"  {q(table_name)} [label={table_label(table_name, tables[table_name])}];")

    source_connectors: list[str] = []
    sink_connectors: list[str] = []
    left_connectors: dict[str, list[str]] = {}
    right_connectors: dict[str, list[str]] = {}
    connector_edges: list[tuple[str, str, dict[str, bool], str, dict[str, dict[str, str | None]]]] = []
    edge_meta: dict[str, dict[str, dict[str, str | None]]] = {}

    for relationship in sorted(
        relationship_list,
        key=lambda rel: (rel["parent"], rel["child"], rel["child_columns"], rel["parent_columns"]),
    ):
        parent = relationship["parent"]
        child = relationship["child"]
        parent_column = relationship["parent_columns"][0] if relationship["parent_columns"] else None
        child_column = relationship["child_columns"][0] if relationship["child_columns"] else None
        parent_in = parent in fragment_tables
        child_in = child in fragment_tables

        if parent_in and child_in:
            constraint = parent != child
            parent_side = MANUAL_INTERNAL_TAIL_SIDES.get(fragment_name, {}).get((parent, child), "e")
            child_side = "e" if parent == child else "w"
            edge_id = edge_id_for(parent, parent_column, child, child_column)
            edge_meta[edge_id] = {
                "tail": edge_endpoint_meta(parent, parent_column),
                "head": edge_endpoint_meta(child, child_column),
            }
            lines.append(
                f"  {endpoint(parent, parent_column, parent_side)} -> {endpoint(child, child_column, child_side)} "
                f"{edge_attrs(constraint=constraint, edge_id=edge_id)};"
            )
            continue

        if parent_in == child_in:
            continue

        connector_id = connector_ids[relationship_key(relationship)]
        connector = connector_node_name(connector_id)
        if parent_in:
            sink_connectors.append(connector)
            right_connectors.setdefault(parent, []).append(connector)
            table_side = connector_table_side(fragment_name, parent, "e")
            connector_side = opposite_side(table_side)
            edge_id = edge_id_for(parent, parent_column, connector, None)
            meta = {
                "tail": edge_endpoint_meta(parent, parent_column),
                "head": edge_endpoint_meta(connector, None),
            }
            edge_meta[edge_id] = meta
            connector_edges.append(
                (endpoint(parent, parent_column, table_side), connector_endpoint(connector, connector_side), {"tailclip": False, "headclip": True}, edge_id, meta)
            )
        else:
            source_connectors.append(connector)
            left_connectors.setdefault(child, []).append(connector)
            table_side = connector_table_side(fragment_name, child, "w")
            connector_side = opposite_side(table_side)
            edge_id = edge_id_for(connector, None, child, child_column)
            meta = {
                "tail": edge_endpoint_meta(connector, None),
                "head": edge_endpoint_meta(child, child_column),
            }
            edge_meta[edge_id] = meta
            connector_edges.append(
                (connector_endpoint(connector, connector_side), endpoint(child, child_column, table_side), {"tailclip": True, "headclip": False}, edge_id, meta)
            )

    connector_names = sorted(set(source_connectors + sink_connectors))
    for connector in connector_names:
        number = int(connector.rsplit("_", 1)[1])
        lines.append(
            f"  {q(connector)} [shape=\"circle\", fixedsize=\"true\", width=\"0.26\", "
            f"height=\"0.26\", margin=\"0\", label={q(number)}, fontsize=\"7\", "
            'fontname="Arial", style="filled", fillcolor="#ffffff", color="#000000", penwidth="1.0"];'
        )

    lines.append("")
    append_rank_groups(lines, fragment["ranks"], left_connectors, right_connectors)
    lines.append("")

    for left, right, attrs, edge_id, _ in connector_edges:
        lines.append(f"  {left} -> {right} {edge_attrs(minlen=1, edge_id=edge_id, **attrs)};")

    lines.append("}")
    lines.append("")
    return "\n".join(lines), edge_meta


def generate() -> list[Path]:
    schema = load_schema()
    created: list[Path] = []
    for fragment_name, fragment in schema["fragments"].items():
        dot_text, edge_meta = render_fragment(schema, fragment_name, fragment)
        dot_path = ROOT / f"{fragment_name}.dot"
        svg_path = ROOT / f"{fragment_name}.svg"

        dot_path.write_text(dot_text, encoding="utf-8")
        subprocess.run(["dot", "-Tsvg", str(dot_path), "-o", str(svg_path)], check=True)
        orthogonalize_svg(svg_path, edge_meta)
        created.extend([dot_path, svg_path])

    return created


def main() -> None:
    created = generate()
    for path in created:
        print(path.relative_to(ROOT.parent.parent))


if __name__ == "__main__":
    main()
