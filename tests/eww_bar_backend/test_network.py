import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class NetworkRadioTests(unittest.TestCase):
    def test_enabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="enabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "true")

    def test_disabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="disabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "false")

    def test_state_includes_wifi_enabled(self):
        with unittest.mock.patch.object(
            collectors, "network_connection_state", return_value={"text": "x", "class": "wifi"}
        ):
            with unittest.mock.patch.object(collectors, "network_radio_enabled", return_value="true"):
                state = collectors.network_state()
        self.assertEqual(state["wifi_enabled"], "true")
        self.assertEqual(state["class"], "wifi")


class WifiToggleTests(unittest.TestCase):
    def _capture(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_toggle_off_when_on(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="true"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "off"]])

    def test_toggle_on_when_off(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="false"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "on"]])

    def test_arg_parsing(self):
        self.assertEqual(
            control.control_payload_from_args(["network", "wifi-toggle"]),
            {"command": "network", "action": "wifi-toggle"},
        )


ROUTE_TEXT = (
    "default via 192.168.0.1 dev eno1 proto dhcp src 192.168.0.88 metric 100\n"
    "192.168.0.0/24 dev eno1 proto kernel scope link src 192.168.0.88 metric 100\n"
)

# Verbatim `ip -o -4 addr show` output, continuation marker included, so a
# future parser rewrite cannot pass here and break against the real command.
ADDR_TEXT = (
    "1: lo    inet 127.0.0.1/8 scope host lo\\       valid_lft forever preferred_lft forever\n"
    "2: eno1    inet 192.168.0.88/24 brd 192.168.0.255 scope global dynamic noprefixroute eno1"
    "\\       valid_lft 41678sec preferred_lft 41678sec\n"
)

# A docked laptop: both links up, each with its own default route. The kernel
# picks the lowest metric, so the wire wins here.
MULTI_ROUTE_TEXT = (
    "default via 192.168.0.1 dev wlan0 proto dhcp src 192.168.0.30 metric 600\n"
    "default via 192.168.0.1 dev eno1 proto dhcp src 192.168.0.88 metric 100\n"
)

NMCLI_STATUS = ("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status")


class LinkFallbackTests(unittest.TestCase):
    def test_finds_default_route_device(self):
        self.assertEqual(collectors.default_route_device(ROUTE_TEXT), "eno1")

    def test_no_default_route(self):
        self.assertEqual(
            collectors.default_route_device("192.168.0.0/24 dev eno1 scope link\n"), ""
        )

    def test_device_ipv4(self):
        self.assertEqual(collectors.device_ipv4(ADDR_TEXT, "eno1"), "192.168.0.88")

    def test_device_ipv4_missing(self):
        self.assertEqual(collectors.device_ipv4(ADDR_TEXT, "wlan0"), "")

    def test_link_state_reports_route_owner(self):
        state = collectors.link_state_from_text(ROUTE_TEXT, ADDR_TEXT)
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])
        self.assertIn("192.168.0.88", state["tooltip"])
        self.assertIn("NetworkManager", state["tooltip"])

    def test_link_state_none_without_default_route(self):
        self.assertIsNone(collectors.link_state_from_text("", ADDR_TEXT))


class NetworkManagerDownTests(unittest.TestCase):
    def _fake_run_text(self, responses):
        def fake(command, **_kwargs):
            return responses.get(tuple(command), "")

        return fake

    def test_falls_back_to_link_state_when_nmcli_is_dead(self):
        responses = {
            NMCLI_STATUS: "",  # explicit: nmcli produced nothing at all
            ("ip", "route"): ROUTE_TEXT,
            ("ip", "-o", "-4", "addr", "show"): ADDR_TEXT,
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])

    def test_disconnected_when_nmcli_dead_and_no_route(self):
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text({})
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "disconnected")

    def test_ethernet_wins_when_it_owns_the_default_route(self):
        responses = {
            ("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"): (
                "wlan0:wifi:connected\neno1:ethernet:connected\n"
            ),
            ("ip", "route"): ROUTE_TEXT,
            ("nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", "eno1"): (
                "IP4.ADDRESS[1]:192.168.0.88/24\n"
            ),
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "ethernet")
        self.assertIn("eno1", state["text"])

    def test_wifi_keeps_selection_when_it_owns_the_default_route(self):
        # Docked laptop on wifi: ethernet is connected but carries no default
        # route, so wifi must stay selected. This is the branch the ethernet
        # short-circuit does not cover.
        responses = {
            NMCLI_STATUS: "wlan0:wifi:connected\neno1:ethernet:connected\n",
            ("ip", "route"): "default via 192.168.0.1 dev wlan0 proto dhcp metric 600\n",
            ("nmcli", "-t", "-f", "GENERAL.CONNECTION,IP4.ADDRESS", "dev", "show", "wlan0"): (
                "GENERAL.CONNECTION:HomeWifi\nIP4.ADDRESS[1]:192.168.0.30/24\n"
            ),
            ("nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi"): "yes:HomeWifi:72\n",
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "wifi")

    def test_falls_back_when_nmcli_reports_nothing_connected(self):
        # NetworkManager is alive but owns nothing (unmanaged devices, or the
        # route belongs to systemd-networkd or a tunnel). The machine is still
        # online, so claiming "Disconnected" would be the same lie.
        responses = {
            NMCLI_STATUS: "eno1:ethernet:unmanaged\nlo:loopback:unmanaged\n",
            ("ip", "route"): ROUTE_TEXT,
            ("ip", "-o", "-4", "addr", "show"): ADDR_TEXT,
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])


class DefaultRouteSelectionTests(unittest.TestCase):
    def test_prefers_the_lowest_metric(self):
        self.assertEqual(collectors.default_route_device(MULTI_ROUTE_TEXT), "eno1")

    def test_route_without_metric_outranks_a_metered_one(self):
        text = "default via 10.0.0.1 dev wlan0 metric 600\ndefault via 10.0.0.1 dev tun0\n"
        self.assertEqual(collectors.default_route_device(text), "tun0")

    def test_truncated_line_does_not_raise(self):
        self.assertEqual(collectors.default_route_device("default via 192.168.0.1 dev\n"), "")

    def test_unparsable_metric_does_not_raise(self):
        text = "default via 10.0.0.1 dev eno1 metric wat\n"
        self.assertEqual(collectors.default_route_device(text), "eno1")


class DegradedReadoutTests(unittest.TestCase):
    def test_text_is_distinguishable_from_healthy_ethernet(self):
        state = collectors.link_state_from_text(ROUTE_TEXT, ADDR_TEXT)
        # Healthy ethernet renders as "<glyph> eno1"; degraded must not look
        # identical to it on the bar.
        self.assertNotEqual(state["text"], "\U000f0317 eno1")

    def test_tooltip_names_the_source_without_overclaiming_the_cause(self):
        state = collectors.link_state_from_text(ROUTE_TEXT, ADDR_TEXT)
        # run_text() flattens "not running", "not on PATH" and "timed out" into
        # the same empty string, so the tooltip must not assert which happened.
        self.assertIn("routing table", state["tooltip"])
        self.assertNotIn("not running", state["tooltip"])


if __name__ == "__main__":
    unittest.main()
