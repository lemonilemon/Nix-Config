import re
import time
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path

from .common import (
    BATTERY_DEFAULT,
    MEDIA_DEFAULT,
    WORKSPACE_DEFAULT,
    idle_pidfile_path,
    parse_json,
    run_text,
    truncate_text,
)


def clock_state():
    now = time.localtime()
    return {
        "time": " " + time.strftime("%H:%M", now),
        "date": " " + time.strftime("%F", now),
        "tooltip": " " + time.strftime("%A, %F %Z", now),
    }


def workspace_state_from_json(active_json, workspaces_json, clients_json):
    active = parse_json(active_json, {})
    workspaces = parse_json(workspaces_json, [])
    clients = parse_json(clients_json, [])
    active_id = active.get("id", 0)

    state = {}
    for workspace_id in range(1, 6):
        occupied = any(
            item.get("id") == workspace_id and item.get("windows", 0) > 0
            for item in workspaces
            if isinstance(item, dict)
        )
        urgent = any(
            item.get("urgent") is True
            and isinstance(item.get("workspace"), dict)
            and item["workspace"].get("id") == workspace_id
            for item in clients
            if isinstance(item, dict)
        )

        if urgent and active_id == workspace_id:
            cls = "active urgent"
            text = ""
        elif urgent:
            cls = "occupied urgent"
            text = ""
        elif active_id == workspace_id:
            cls = "active"
            text = ""
        elif occupied:
            cls = "occupied"
            text = ""
        else:
            cls = "empty"
            text = ""

        state[f"ws{workspace_id}_class"] = cls
        state[f"ws{workspace_id}_text"] = text

    return state


def workspace_state():
    return workspace_state_from_json(
        run_text(["hyprctl", "activeworkspace", "-j"]),
        run_text(["hyprctl", "workspaces", "-j"]),
        run_text(["hyprctl", "clients", "-j"]),
    )


def submap_from_event(line):
    if not line.startswith("submap>>"):
        return None
    value = line.split(">>", 1)[1]
    if value in ("default", "reset"):
        return ""
    return value


def media_state_from_text(status_text, metadata_text):
    metadata = metadata_text.splitlines()
    if not metadata or not metadata[0].strip():
        return MEDIA_DEFAULT.copy()

    status_lines = status_text.splitlines()
    status = status_lines[0].strip() if status_lines else ""
    normalized = status.lower()
    if normalized == "playing":
        icon = ""
        cls = "playing"
        display_status = "Playing"
    elif normalized == "paused":
        icon = ""
        cls = "paused"
        display_status = "Paused"
    elif normalized == "stopped":
        icon = ""
        cls = "stopped"
        display_status = "Stopped"
    else:
        icon = ""
        cls = ""
        display_status = status or "Media"

    return {
        "text": truncate_text(metadata[0].strip(), 52),
        "status": display_status,
        "icon": icon,
        "class": cls,
        "tooltip": f"{display_status}\nLeft click: play/pause\nRight click: next",
    }


def media_state():
    return media_state_from_text(
        run_text(["playerctl", "status"], timeout=1.0),
        run_text(["playerctl", "metadata", "--format", "{{artist}} - {{title}}"]),
    )


def cpu_state():
    def read_cpu():
        parts = Path("/proc/stat").read_text().splitlines()[0].split()
        values = [int(value) for value in parts[1:8]]
        idle = values[3] + values[4]
        total = sum(values)
        return total, idle

    try:
        total1, idle1 = read_cpu()
        time.sleep(0.2)
        total2, idle2 = read_cpu()
        total_delta = total2 - total1
        idle_delta = idle2 - idle1
        if total_delta <= 0:
            return " --%"
        usage = (total_delta - idle_delta) * 100 / total_delta
        return f" {usage:.0f}%"
    except Exception:
        return " --%"


def memory_state_from_text(text):
    values = {}
    for line in text.splitlines():
        parts = line.split()
        if len(parts) >= 2:
            values[parts[0].rstrip(":")] = int(parts[1])

    total = values.get("MemTotal", 0)
    available = values.get("MemAvailable", 0)
    swap_total = values.get("SwapTotal", 0)
    swap_free = values.get("SwapFree", 0)
    if total <= 0:
        return {"text": " --%", "tooltip": "", "class": ""}

    pct = (total - available) / total * 100
    used_g = (total - available) / 1048576
    total_g = total / 1048576
    swap_used_g = (swap_total - swap_free) / 1048576
    swap_total_g = swap_total / 1048576
    cls = "critical" if pct >= 80 else ""
    return {
        "text": f" {pct:.0f}%",
        "tooltip": f"Used: {used_g:.1f}G/{total_g:.1f}G\nSwap: {swap_used_g:.1f}G/{swap_total_g:.1f}G",
        "class": cls,
    }


def memory_state():
    try:
        return memory_state_from_text(Path("/proc/meminfo").read_text())
    except Exception:
        return {"text": " --%", "tooltip": "", "class": ""}


def temperature_state():
    max_temp = None
    for input_path in Path("/sys/class/hwmon").glob("hwmon*/temp*_input"):
        try:
            temp = int(input_path.read_text().strip()) / 1000
        except Exception:
            continue
        if max_temp is None or temp > max_temp:
            max_temp = temp

    if max_temp is None:
        return {"text": " --°C", "class": ""}
    rounded = int(round(max_temp))
    return {
        "text": f" {rounded}°C",
        "class": "critical" if rounded >= 90 else "",
    }


def first_ip(ip_text):
    for line in ip_text.splitlines():
        if ":" in line:
            return line.split(":", 1)[1]
    return ""


def nmcli_value(text, key):
    for line in text.splitlines():
        if line.startswith(key + ":"):
            return line.split(":", 1)[1]
    return ""


def wireless_signal_percent(wireless_text, interface):
    for line in wireless_text.splitlines():
        stripped = line.strip()
        if not stripped.startswith(interface + ":"):
            continue
        parts = stripped.replace(":", " ").split()
        if len(parts) < 3:
            return None
        try:
            quality = float(parts[2].rstrip("."))
        except ValueError:
            return None
        return max(0, min(100, int(round(quality * 100 / 70))))
    return None


def connected_device(status_text, device_type):
    for line in status_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[1] == device_type and fields[2] == "connected":
            return fields[0]
    return ""


def network_state_from_text(status_text, wifi_text, ip_by_device):
    wifi_device = ""
    for line in status_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[1] == "wifi" and fields[2] == "connected":
            wifi_device = fields[0]
            break

    wifi = ""
    for line in wifi_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[0] == "yes":
            wifi = line
            break

    if wifi_device and wifi:
        fields = wifi.split(":")
        ssid = fields[1]
        signal = fields[2]
        ip_info = first_ip(ip_by_device.get(wifi_device, ""))
        if ip_info:
            return {
                "text": f" {ssid} {signal}%",
                "tooltip": f"{wifi_device}: {ip_info}",
                "class": "wifi",
            }
        return {
            "text": "󰈀 (No IP)",
            "tooltip": f"{wifi_device}: No IP",
            "class": "linked",
        }

    for line in status_text.splitlines():
        fields = line.split(":")
        if len(fields) >= 3 and fields[1] == "ethernet" and fields[2] == "connected":
            device = fields[0]
            ip_info = first_ip(ip_by_device.get(device, ""))
            if ip_info:
                return {
                    "text": f"󰌗 {device}",
                    "tooltip": f"{device}: {ip_info}",
                    "class": "ethernet",
                }
            return {
                "text": "󰈀 (No IP)",
                "tooltip": f"{device}: No IP",
                "class": "linked",
            }

    return {
        "text": "⚠ Disconnected",
        "tooltip": "No connection",
        "class": "disconnected",
    }


def network_state():
    status = run_text(["nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"])
    wifi_device = connected_device(status, "wifi")
    if wifi_device:
        details = run_text(
            ["nmcli", "-t", "-f", "GENERAL.CONNECTION,IP4.ADDRESS", "dev", "show", wifi_device]
        )
        ssid = nmcli_value(details, "GENERAL.CONNECTION") or wifi_device
        ip_info = first_ip(details)
        try:
            wireless_text = Path("/proc/net/wireless").read_text()
        except Exception:
            wireless_text = ""
        signal = wireless_signal_percent(wireless_text, wifi_device)
        if signal is None:
            wifi = run_text(["nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi"])
            return network_state_from_text(status, wifi, {wifi_device: details})
        if ip_info:
            return {
                "text": f" {ssid} {signal}%",
                "tooltip": f"{wifi_device}: {ip_info}",
                "class": "wifi",
            }
        return {
            "text": "󰈀 (No IP)",
            "tooltip": f"{wifi_device}: No IP",
            "class": "linked",
        }

    ethernet_device = connected_device(status, "ethernet")
    if ethernet_device:
        details = run_text(["nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", ethernet_device])
        ip_info = first_ip(details)
        if ip_info:
            return {
                "text": f"󰌗 {ethernet_device}",
                "tooltip": f"{ethernet_device}: {ip_info}",
                "class": "ethernet",
            }
        return {
            "text": "󰈀 (No IP)",
            "tooltip": f"{ethernet_device}: No IP",
            "class": "linked",
        }

    return {
        "text": "⚠ Disconnected",
        "tooltip": "No connection",
        "class": "disconnected",
    }


def volume_state_from_text(text):
    match = re.search(r"Volume:\s+([0-9.]+)", text)
    if not match:
        return ""
    volume = int((Decimal(match.group(1)) * 100).to_integral_value(rounding=ROUND_HALF_UP))
    if "[MUTED]" in text:
        return "󰝟"
    if volume < 35:
        return f" {volume}%"
    if volume < 70:
        return f" {volume}%"
    return f" {volume}%"


def volume_state():
    return volume_state_from_text(run_text(["wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"]))


def battery_state():
    for battery in Path("/sys/class/power_supply").glob("BAT*"):
        capacity_path = battery / "capacity"
        if not capacity_path.exists():
            continue
        try:
            capacity = int(capacity_path.read_text().strip())
        except Exception:
            continue
        try:
            status = (battery / "status").read_text().strip()
        except Exception:
            status = ""

        if status == "Charging":
            icon = "󰂄"
        elif capacity < 20:
            icon = "󰁻"
        elif capacity < 40:
            icon = "󰁼"
        elif capacity < 60:
            icon = "󰁾"
        elif capacity < 80:
            icon = "󰂀"
        else:
            icon = "󰁹"

        time_str = "N/A"
        now = rate = full_value = None
        for now_name, rate_name, full_name in [
            ("energy_now", "power_now", "energy_full"),
            ("charge_now", "current_now", "charge_full"),
        ]:
            try:
                now = int((battery / now_name).read_text().strip())
                rate = int((battery / rate_name).read_text().strip())
                full_value = int((battery / full_name).read_text().strip())
                break
            except Exception:
                now = rate = full_value = None

        minutes = None
        if now is not None and rate and rate > 0:
            if status == "Discharging":
                minutes = now * 60 // rate
            elif status == "Charging" and full_value:
                minutes = (full_value - now) * 60 // rate
        if minutes is not None:
            time_str = f"{minutes // 60}h {minutes % 60}m"

        cls = ""
        if status == "Charging":
            cls = "charging"
        elif capacity < 20:
            cls = "critical"
        elif capacity < 30:
            cls = "warning"
        return {
            "text": f"{icon} {capacity}%",
            "alt": f"{icon} {time_str}",
            "capacity": capacity,
            "class": cls,
        }
    return BATTERY_DEFAULT.copy()


def parse_controller(controller_text):
    address = ""
    alias = ""
    powered = "off"
    for line in controller_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("Controller "):
            parts = stripped.split()
            if len(parts) >= 2:
                address = parts[1]
        elif stripped.startswith("Alias:"):
            alias = stripped.split(":", 1)[1].strip()
        elif stripped.startswith("Powered:"):
            powered = stripped.split(":", 1)[1].strip()
    return alias or "Bluetooth", address or "N/A", powered


def parse_device_info(info_text, fallback_alias):
    alias = fallback_alias
    battery = ""
    for line in info_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("Alias:"):
            alias = stripped.split(":", 1)[1].strip()
        match = re.search(r"Battery Percentage:.*\((\d+)\)", stripped)
        if match:
            battery = match.group(1) + "%"
    return alias, battery


def bluetooth_state_from_text(controller_text, devices_text, info_by_address):
    controller_alias, controller_address, powered = parse_controller(controller_text)
    devices = [line for line in devices_text.splitlines() if line.strip()]
    if not devices:
        tooltip = f"{controller_alias}\t{controller_address}\n\n0 connected"
        return {"text": "", "tooltip": tooltip, "class": powered}

    preferred = next(
        (line for line in devices if re.search(r"ugreen_1|ugreen_2", line, re.I)),
        devices[0],
    )
    preferred_parts = preferred.split(maxsplit=2)
    first_address = preferred_parts[1] if len(preferred_parts) >= 2 else ""
    fallback_alias = preferred_parts[2] if len(preferred_parts) >= 3 else first_address
    first_alias, first_battery = parse_device_info(
        info_by_address.get(first_address, ""), fallback_alias
    )
    text = f"󰂯 {first_alias} {first_battery}" if first_battery else f"󰂯 {first_alias}"
    text = truncate_text(text, 24)

    detail_lines = []
    for line in devices:
        parts = line.split(maxsplit=2)
        if len(parts) < 2:
            continue
        address = parts[1]
        fallback = parts[2] if len(parts) >= 3 else address
        alias, battery = parse_device_info(info_by_address.get(address, ""), fallback)
        if battery:
            detail_lines.append(f"{alias}\t{address}\t{battery}")
        else:
            detail_lines.append(f"{alias}\t{address}")

    tooltip = (
        f"{controller_alias}\t{controller_address}\n\n"
        f"{len(devices)} connected\n\n"
        + "\n".join(detail_lines)
    )
    return {"text": text, "tooltip": tooltip, "class": "connected"}


def bluetooth_state():
    controller = run_text(["bluetoothctl", "show"])
    devices = run_text(["bluetoothctl", "devices", "Connected"])
    info_by_address = {}
    for line in devices.splitlines():
        parts = line.split(maxsplit=2)
        if len(parts) >= 2:
            info_by_address[parts[1]] = run_text(["bluetoothctl", "info", parts[1]])
    return bluetooth_state_from_text(controller, devices, info_by_address)


def idle_inhibited_state():
    pidfile = idle_pidfile_path()
    try:
        pid = int(pidfile.read_text().strip())
    except Exception:
        return "false"
    return "true" if Path(f"/proc/{pid}").exists() else "false"
