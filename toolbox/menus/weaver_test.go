package menus

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/runner-mei/log"
	"github.com/runner-mei/log/logtest"
	"github.com/runner-mei/moo/moo_tests"
)

func TestMenuSimple(t *testing.T) {
	env := moo_tests.NewEnvironment(t, nil)
	sendEvent := func(descr string) {}

	for tidx, test := range tests {
		logger := log.New(os.Stderr)
		weaver, err := NewWeaver(logger, env, sendEvent, map[string]Layout{
			"default": test.layout,
		})
		if err != nil {
			t.Error(tidx, test.name, err)
			return
		}
		for idx, step := range test.steps {
			t.Run(step.name, func(t *testing.T) {
				if step.isRestart {
					weaver, err = NewWeaver(logger, env, sendEvent, map[string]Layout{
						"default": test.layout,
					})
					if err != nil {
						t.Error(tidx, test.name, err)
						return
					}
				}

				if err := weaver.Update(step.app, step.value); err != nil {
					t.Error("[", tidx, test.name, "] [", idx, step, "]", err)
					return
				}

				results, err := weaver.Generate("default")
				if err != nil {
					return
				}

				if !IsSameMenuArray(results, step.results()) {

					msg := cmp.Diff(step.results(), results)
					t.Error(msg)

					// bs, _ := json.MarshalIndent(weaver.Stats(), "", "  ")
					// t.Log(string(bs))
				}
			})
		}
	}
}

type testStep struct {
	isRestart bool
	name      string
	app       string
	value     []Menu
	results   func() []Menu
}

var test2 = []testStep{
	{
		name:  "app1",
		app:   "app1",
		value: testapp1,
		results: func() []Menu {
			results := Copy(menuTestLayout)
			list := searchMenuListInTree(results, "1", nil)
			*list[0] = testapp1[0]
			list = searchMenuListInTree(results, "3", nil)
			*list[0] = testapp1[1]
			return results
		},
	},
	{
		name:  "app2",
		app:   "app2",
		value: testapp2,
		results: func() []Menu {
			results := Copy(menuTestLayout)
			list := searchMenuListInTree(results, "1", nil)
			*list[0] = testapp1[0]
			list = searchMenuListInTree(results, "3", nil)
			*list[0] = testapp1[1]

			list = searchMenuListInTree(results, "2", nil)
			*list[0] = testapp2[0]
			list = searchMenuListInTree(results, "4", nil)
			*list[0] = testapp2[1]

			list = searchMenuListInTree(results, "4_1", nil)
			for _, to := range list {
				*to = testapp2[1].Children[0]
			}
			return results
		},
	},
	{
		name:  "app3",
		app:   "app3",
		value: testapp3,
		results: func() []Menu {
			return Copy(menuTestResults)
		},
	},
}

var tests = []struct {
	layout Layout
	name   string
	steps  []testStep
}{
	{
		name: "a2",
		layout: NewSimple(
			logtest.New(),
			Copy(menuTestLayout),
			func(Menu) bool {
				return true
			}),
		steps: test2,
	},
}

// func TestMenuSimple2(t *testing.T) {
// 	env := env_tests.Clone(nil)

// 	for _, test := range []struct {
// 		filename string
// 		layout   string
// 	}{
// 		{filename: "tests\\test1.json", layout: "am"},
// 		{filename: "tests\\test2.json"},
// 		{filename: "tests\\test3.json", layout: "vpn_management"},
// 	} {
// 		t.Run(test.filename, func(t *testing.T) {
// 			// core, _ := engine.NewCore(&engine.Options{})

// 			logger := log.New(os.Stderr)
// 			weaver := &menuWeaver{Logger: logger, env: env, sendEvent: func(hub.Message) {
// 				// core.CreateTopicIfNotExists()
// 			}, layouts: nil}

// 			data, err := ioutil.ReadFile(test.filename)
// 			if err != nil {
// 				t.Error(err)
// 				return
// 			}

// 			var udata = &struct {
// 				Apps    map[string][]Menu `json:"applications"`
// 				Layouts map[string][]LayoutItem   `json:"layout"`
// 				Results []Menu            `json:"menuList"`
// 			}{}
// 			err = json.Unmarshal(data, &udata)
// 			if err != nil {
// 				t.Error(err)
// 				return
// 			}

// 			weaver.byApplications = udata.Apps
// 			weaver.layouts = map[string]Layout{}
// 			for key, value := range udata.Layouts {
// 				weaver.layouts[key] = &layoutImpl{value}
// 			}

// 			weaver.layout = weaver.layouts["default"]

// 			if err := weaver.Init(); err != nil {
// 				t.Error(err)
// 				return
// 			}
// 			weaver.byApplications = udata.Apps
// 			weaver.menuList = nil
// 			weaver.menuListByLayout = map[string][]Menu{}

// 			results, err := weaver.Generate(test.layout)
// 			if err != nil {
// 				t.Error(err)
// 				return
// 			}

// 			if !isSameMenuArray(results, udata.Results) {
// 				t.Error("not same")
// 				var s1, s2 strings.Builder
// 				FormatMenus(&s1, nil, udata.Results, 0, true)
// 				FormatMenus(&s2, nil, results, 0, true)

// 				seq1 := strings.Split(s1.String(), "\n")
// 				seq2 := strings.Split(s2.String(), "\n")
// 				records := difflib.Diff(seq1, seq2)
// 				for idx := range records {
// 					t.Log(records[idx])
// 				}

// 				// FormatMenus(os.Stdout, nil, results, 0, true)

// 				return
// 			}
// 		})
// 	}
// }

// func TestMenuSimple3(t *testing.T) {
// 	// env := env_tests.Clone(nil)
// 	txt := `[{"uid":"nm.home","title":"运维主页","permission":"nm.home","url":"/hengwei/web/network_topologies/root","icon":"fa-home"},{"uid":"nm.monitor1","title":"总览视图","icon":"fa-trello","children":[{"uid":"nm.monitor.topology","title":"网络视图","permission":"nm.monitor.topology","url":"/hengwei/web/network_topologies/root"},{"uid":"nm.monitor.computer_room","title":"机房视图","permission":"nm.monitor.computer_room","url":"/hengwei/web/room_topologies/first"},{"uid":"nm.monitor.biz_view_list","title":"业务视图","permission":"nm.monitor.biz_view_list","url":"/hengwei/web/biz_management/biz_views"},{"uid":"nm.monitor.divider1","title":"divider","permission":"nm.monitor.divider1"},{"uid":"nm.monitor.view_panel","title":"监控总览","permission":"nm.monitor.view_panel","url":"/hengwei/web/integrated_monitor"},{"uid":"nm.monitor.biz_browse","title":"业务总览","permission":"nm.monitor.biz_browse","url":"/hengwei/web/biz_management/biz_overview"},{"uid":"nm.monitor.services_browse","title":"服务总览","permission":"nm.monitor.services_browse","url":"/hengwei/web/services/tree_view"},{"uid":"nm.monitor.device_browse","title":"设备总览","permission":"nm.monitor.device_browse","url":"/hengwei/web/network_devices/device_browse"}]},{"uid":"nm.auto.operation1","title":"智动运维","permission":"nm.auto_operation","icon":"fa-gears","children":[{"uid":"mc.Todolists.Index","title":"待办事项","permission":"mc.Todolists","url":"/hengwei/mc/todolist"}]},{"uid":"nm.data_flow1","title":"实时数据分析","icon":"fa-clock-o","children":[{"uid":"nm.monitor.all_view","title":"实时数据曲线","permission":"nm.monitor.all_view","url":"/hengwei/web/monitor_browse?mo_type=\u0026mo_id=\u0026status="},{"uid":"nm.monitor_management.device_monitor","title":"设备负载监控","permission":"nm.monitor_management.device_monitor","children":[{"uid":"nm.monitor_management.device_cpu_monitor","title":"CPU利用率","permission":"nm.monitor_management.device_cpu_monitor","url":"/hengwei/web/mos/view_status_by_select?managed_type=network_device\u0026metric=cpu\u0026attribute=cpu","icon":"show-remote"},{"uid":"nm.monitor_management.device_mem_monitor","title":"内存利用率","permission":"nm.monitor_management.device_mem_monitor","url":"/hengwei/web/mos/view_status_by_select?managed_type=network_device\u0026metric=mem\u0026attribute=used_per","icon":"show-remote"}]},{"uid":"nm.monitor_management.line_monitor","title":"线路流量监控","permission":"nm.monitor_management.line_monitor","children":[{"uid":"nm.monitor_management.line_octets_monitor","title":"上下行流量","permission":"nm.monitor_management.line_octets_monitor","url":"/hengwei/web/mos/view_overlap_status_by_select?managed_type=network_link\u0026up_status=link_flux%24ifOutOctets\u0026down_status=link_flux%24ifInOctets\u0026title=%E4%B8%8A%E4%B8%8B%E8%A1%8C%E6%B5%81%E9%87%8F","icon":"show-remote"},{"uid":"nm.monitor_management.line_pkts_monitor","title":"上下行帧流量","permission":"nm.monitor_management.line_pkts_monitor","url":"/hengwei/web/mos/view_overlap_status_by_select?managed_type=network_link\u0026up_status=link_flux%24ifOutUcastPkts\u0026down_status=link_flux%24ifInUcastPkts\u0026title=%E4%B8%8A%E4%B8%8B%E8%A1%8C%E5%B8%A7%E6%B5%81%E9%87%8F","icon":"show-remote"}]},{"uid":"nm.monitor_management.topn","title":"负载TOPN","children":[{"uid":"nm.monitor_management.device_cpu_topn","title":"CPU利用率TOP-N","permission":"nm.monitor_management.device_cpu_topn","url":"/hengwei/web/monitor_browse/device_load_topn/cpuLoad","icon":"show-remote"},{"uid":"nm.monitor_management.device_mem_topn","title":"内存利用率TOP-N","permission":"nm.monitor_management.device_mem_topn","url":"/hengwei/web/monitor_browse/device_load_topn/memUsage","icon":"show-remote"},{"uid":"nm.monitor_management.divider","title":"divider"},{"uid":"nm.monitor_management.link_octets_topn","title":"线路流量TOP-N","permission":"nm.monitor_management.link_octets_topn","url":"/hengwei/web/monitor_browse/link_load_topn/ifOctets","icon":"show-remote"},{"uid":"nm.monitor_management.link_pkts_topn","title":"帧流量TOP-N","permission":"nm.monitor_management.link_pkts_topn","url":"/hengwei/web/monitor_browse/link_load_topn/ifUcastPkts","icon":"show-remote"}]},{"uid":"nm.device.table","title":"设备表信息","children":[{"uid":"nm.monitor_management.arp_info","title":"设备ARP表信息","permission":"nm.monitor_management.arp_info","url":"/hengwei/web/mos/view_metric_by_select?managed_type=network_device\u0026metric=arp","icon":"show-remote"},{"uid":"nm.monitor_management.address_info","title":"设备IP地址表信息","permission":"nm.monitor_management.address_info","url":"/hengwei/web/mos/view_metric_by_select?managed_type=network_device\u0026metric=ip_address","icon":"show-remote"},{"uid":"nm.monitor_management.tcp_connection","title":"设备TCP连接信息","permission":"nm.monitor_management.tcp_connection","url":"/hengwei/web/mos/view_metric_by_select?managed_type=network_device\u0026metric=tcp_connection","icon":"show-remote"},{"uid":"nm.monitor_management.udp_listen","title":"设备UDP监听信息","permission":"nm.monitor_management.udp_listen","url":"/hengwei/web/mos/view_metric_by_select?managed_type=network_device\u0026metric=udp_listen","icon":"show-remote"}]},{"uid":"nm.view.runtime","title":"运行视图","children":[{"uid":"nm.monitor_management.device_view","title":"设备运行视图","permission":"nm.monitor_management.device_view","url":"/hengwei/web/mos/view_monitor_by_select?managed_type=network_device","icon":"show-remote"},{"uid":"nm.monitor_management.host_view","title":"主机运行视图","permission":"nm.monitor_management.host_view","url":"/hengwei/web/mos/view_monitor_by_select?managed_type=service_host","icon":"show-remote"},{"uid":"nm.monitor_management.db_view","title":"数据库运行视图","permission":"nm.monitor_management.db_view","url":"/hengwei/web/mos/view_monitor_by_select?managed_type=service_database","icon":"show-remote"},{"uid":"nm.monitor_management.mid_view","title":"中间件运行视图","permission":"nm.monitor_management.mid_view","url":"/hengwei/web/mos/view_monitor_by_select?managed_type=service_midware","icon":"show-remote"}]}]},{"uid":"nm.common_tools1","title":"常用工具","icon":"fa-wrench","children":[{"uid":"nm.common_tools.ping_test","title":"Ping测试","permission":"nm.common_tools.ping_test","url":"/hengwei/web/tools/shell_modal?cmd=ping\u0026address=","icon":"show-remote"},{"uid":"nm.common_tools.snmp_test","title":"Snmp连接测试","permission":"nm.common_tools.snmp_test","url":"/hengwei/web/tools/shell_modal?cmd=snmpgetnext\u0026address=","icon":"show-remote"},{"uid":"nm.common_tools.trace_route","title":"Trace Route","permission":"nm.common_tools.trace_route","url":"/hengwei/web/tools/shell_modal?cmd=tracert\u0026address=","icon":"show-remote"},{"uid":"nm.common_tools.dig_test","title":"Dig","permission":"nm.common_tools.dig_test","url":"/hengwei/web/tools/shell_modal?cmd=runtime_env%2Fdig%2Fdig\u0026address=","icon":"show-remote"},{"uid":"nm.common_tools.divider3","title":"divider","permission":"nm.common_tools.divider3","icon":"show-remote"},{"uid":"nm.common_tools.telnet","title":"Telnet","permission":"nm.common_tools.telnet","url":"/hengwei/internal/web/terminal/index.html?protocol=telnet"},{"uid":"nm.common_tools.ssh","title":"SSH","permission":"nm.common_tools.ssh","url":"/hengwei/internal/web/terminal/index.html?protocol=ssh"},{"uid":"nm.common_tools.divider4","title":"divider","permission":"nm.common_tools.divider4"},{"uid":"nm.common_tools.cim","title":"Cim浏览","permission":"nm.common_tools.cim","url":"/hengwei/web/tools/cim_navigator"},{"uid":"nm.common_tools.divider5","title":"divider","permission":"nm.common_tools.divider5"},{"uid":"nm.ipDevice.distribution","title":"IP设备分布","permission":"nm.ipDevice.distribution","url":"/hengwei/web/system_settings/ip_device_distribution"},{"uid":"nm.ipService.distribution","title":"IP服务分布","permission":"nm.ipService.distribution","url":"/hengwei/web/system_settings/ip_service_distribution"}]},{"uid":"nm.alarm_log1","title":"告警\u0026日志","icon":"fa-bell","children":[{"uid":"nm.alert.current_alert","title":"未恢复告警","permission":"nm.alert.current_alert","url":"/hengwei/web/alert_events/list?source=cookies","icon":"show-remote"},{"uid":"nm.log","title":"日志信息","icon":"fa-log","children":[{"uid":"nm.alert.log","title":"告警日志","permission":"nm.alert.log","url":"/hengwei/web/alert_events?source=histories"},{"uid":"nm.alert.device_port_log","title":"设备端口日志","permission":"nm.alert.device_port_log","url":"/hengwei/web/snmp_traps/for_devices"},{"uid":"nm.alert.composite_log","title":"组合告警日志","permission":"nm.alert.composite_log","url":"/hengwei/web/composite_alert_events?source=histories"},{"uid":"nm.notifications.list","title":"外部日志告警日志","permission":"nm.notifications.list","url":"/hengwei/web/notifications/confirmed"},{"uid":"nm.harvest.trap_query","title":"SnmpTrap日志","permission":"nm.harvest.trap_query","url":"/hengwei/web/snmp_traps"},{"uid":"nm.alarm_log.divider1","title":"divider"},{"uid":"nm.alert.error_list","title":"告警规则运行日志","permission":"nm.alert.error_list","url":"/hengwei/web/alarm_rules/error_list"},{"uid":"nm.report_management.history_error_list","title":"历史规则运行日志","permission":"nm.report_management.history_error_list","url":"/hengwei/web/histories/error_list"},{"uid":"nm.sms.list","title":"短信发出日志","permission":"nm.sms.list","url":"/hengwei/web/sms_management/sms_view"},{"uid":"nm.common_tools.error_list","title":"颜色规则日志","permission":"nm.common_tools.error_list","url":"/hengwei/web/system/redis_command_errors"}]},{"uid":"loganalyzer","title":"Syslog日志","children":[{"uid":"loganalyzer.dashboards","title":"总览","permission":"nm.harvest.syslog_query","url":"/hengwei/loganalyzer/views/dashboards.html"},{"uid":"loganalyzer.view","title":"查询","permission":"nm.harvest.syslog_query","url":"/hengwei/loganalyzer/views/index.html"},{"uid":"loganalyzer.filters","title":"过滤器","permission":"nm.harvest.syslog_query","url":"/hengwei/loganalyzer/views/filters/index.html"},{"uid":"loganalyzer.settings","title":"设置","permission":"nm.harvest.syslog_settings","url":"/hengwei/loganalyzer/views/settings.html"}]},{"uid":"nm.alarm_rule","title":"规则设置","icon":"fa-alarm_rule","children":[{"uid":"nm.alert.rule","title":"告警规则","permission":"nm.alert.rule","url":"/hengwei/web/alarm_rules"},{"uid":"nm.alert.composited_rule","title":"组合告警","permission":"nm.alert.composited_rule","url":"/hengwei/web/alarm_composited_rules"},{"uid":"nm.alarm_rule.divider3","title":"divider"},{"uid":"nm.alert.log_alert","title":"外部日志告警规则","permission":"nm.alert.log_alert","url":"/hengwei/web/log_alarm_rules"},{"uid":"nm.alert.custom_detect_rule","title":"自定义检测告警","permission":"nm.alert.custom_detect_rule","url":"/hengwei/web/custom_inpect_alert_rules"},{"uid":"nm.alarm_rule.divider4","title":"divider"},{"uid":"nm.alert.baseline","title":"基线数据管理","permission":"nm.alert.baseline","url":"/hengwei/web/baselines"}]}]},{"uid":"nm.resource1","title":"管理对象","icon":"fa-hdd-o","children":[{"uid":"nm.topology.search","title":"生成拓扑图","permission":"nm.topology.search","url":"/hengwei/web/topologies/search_params"},{"uid":"nm.topology.report","title":"拓扑生成报告","permission":"nm.topology.report","url":"/hengwei/web/topologies/search_report"},{"uid":"xxxxxx","title":"运维平台","permission":"products.wserver","url":"/hengwei/","icon":"fa-trello"},{"uid":"app.products","title":"综合运维中心","icon":"fa-desktop","children":[{"uid":"product-wserver","title":"运维平台","permission":"products.wserver","url":"/hengwei/","icon":"fa-trello"},{"uid":"product-am","title":"IT资产管理","permission":"products.am","url":"/hengwei/am","icon":"fa-archive"},{"uid":"product-itsm","title":"IT服务管理","permission":"products.itsm","url":"/hengwei/itsm","icon":"fa-asterisk"}]},{"uid":"nm.system.help2","title":"帮助","url":"/hengwei/internal/doc/"},{"uid":"nm.resource.device","title":"设备管理","permission":"nm.resource.device","url":"/hengwei/web/network_devices"},{"uid":"nm.resource.link","title":"线路管理","permission":"nm.resource.link","url":"/hengwei/web/network_links"},{"uid":"nm.resource.composite_port","title":"组合端口管理","permission":"nm.resource.composite_port","url":"/hengwei/web/composite_ports"},{"uid":"nm.resource.divider1","title":"divider","permission":"nm.resource.divider1"},{"uid":"nm.resource.host","title":"主机管理","permission":"nm.resource.host","url":"/hengwei/web/service_hosts"},{"uid":"nm.resource.database","title":"数据库管理","permission":"nm.resource.database","url":"/hengwei/web/service_databases"},{"uid":"nm.resource.midware","title":"中间件管理","permission":"nm.resource.midware","url":"/hengwei/web/service_midwares"},{"uid":"nm.resource.app","title":"标准应用管理","permission":"nm.resource.app","url":"/hengwei/web/std_services"},{"uid":"nm.resource.divider3","title":"divider","permission":"nm.resource.divider3"},{"uid":"nm.resource.tag_list","title":"Tag列表","permission":"nm.resource.tag_list","url":"/hengwei/web/mos/tags"},{"uid":"nm.resource.mo_setting","title":"管理对象设置","permission":"nm.resource.mo_setting","url":"/hengwei/web/mos/all_settings"},{"uid":"nm.resource.run_time","title":"运行时段设置","permission":"nm.resource.run_time","url":"/hengwei/web/system_settings/mo_run_time"},{"uid":"nm.resource.divider4","title":"divider","permission":"nm.resource.divider4"},{"uid":"nm.resource.import","title":"资源导入","permission":"nm.resource.import","url":"/hengwei/web/system_settings/resource_import"},{"uid":"nm.resource.diagnosis","title":"资源诊断","permission":"nm.resource.diagnosis","url":"/hengwei/web/system_settings/resource_diagnosis"}]},{"uid":"nm.system2","title":"系统设置","icon":"fa-cog","children":[{"uid":"um.HengweiUsers","title":"用户管理","permission":"um.HengweiUsers","url":"/hengwei/um/hengweiusers/index"},{"uid":"um.HengweiUserGroups","title":"用户组管理","permission":"um.HengweiUserGroups","url":"/hengwei/um/hengweiusergroups/index?groupId=0"},{"uid":"um.HengweiRoles","title":"角色管理","permission":"um.HengweiRoles","url":"/hengwei/um/hengweiroles/index"},{"uid":"um.HengweiPermissionGroups","title":"权限管理","permission":"um.HengweiPermissionGroups","url":"/hengwei/um/hengweipermissiongroups/index?groupId=0\u0026tagID="},{"uid":"nm.system.divider2","title":"divider","permission":"nm.system.divider2"},{"uid":"mc.SchdJobs.Index","title":"计划任务","permission":"mc.SchdJobs","url":"/hengwei/mc/schdjobs/index"},{"uid":"nm.system.engine_nodes","title":"分布式采集引擎","permission":"nm.system.engine_nodes","url":"/hengwei/web/system/engine_nodes"},{"uid":"nm.system.mail_server","title":"外部服务器设置","permission":"nm.system.mail_server","url":"/hengwei/web/system_settings/mail_server"},{"uid":"nm.system.divider3","title":"divider","permission":"nm.system.divider3"},{"uid":"nm.system.image_manage","title":"图元管理","permission":"nm.system.image_manage","url":"/hengwei/web/custom_resources/image_view?path=images%2Ftopology"},{"uid":"nm.system.script_manage","title":"脚本管理","permission":"nm.system.script_manage","url":"/hengwei/web/tools/script_manage"},{"uid":"nm.system.sql_script_manage","title":"数据库脚本管理","permission":"nm.system.sql_script_manage","url":"/hengwei/web/tools/sql_script_manage"},{"uid":"nm.system.divider4","title":"divider","permission":"nm.system.divider4"},{"uid":"nm.system.operation_logs","title":"用户操作日志","permission":"nm.system.operation_logs","url":"/hengwei/web/system/operation_logs"},{"uid":"nm.system.divider5","title":"divider","permission":"nm.system.divider5"},{"uid":"mc.NotificationGroups.Index","title":"通知管理","permission":"mc.NotificationGroups","url":"/hengwei/mc/notificationgroups/index"},{"uid":"nm.system.divider6","title":"divider","permission":"nm.system.divider6"},{"uid":"mc.Tables.Index","title":"系统空间管理","permission":"mc.Tables","url":"/hengwei/mc/tables/index?active="},{"uid":"mc.TSDBShards.Index","title":"历史数据管理","permission":"mc.TSDBShards","url":"/hengwei/mc/tsdbshards/index"},{"uid":"mc.AutoBackup.Index","title":"备份数据","permission":"mc.AutoBackup","url":"/hengwei/mc/autobackup/index?active="},{"uid":"nm.system.divider7","title":"divider"},{"uid":"nm.system.generate_license","title":"生成License","permission":"nm.system.generate_license","url":"/hengwei/internal/wserver/license"},{"uid":"nm.system.import_license","title":"导入License","permission":"nm.system.import_license","url":"/hengwei/internal/wserver/import_license"},{"uid":"nm.system.divider8","title":"divider"},{"uid":"nm.system.help","title":"帮助"},{"uid":"mc.About.Index","title":"关于","permission":"mc.About","url":"/hengwei/mc/about/index"}]}]`

// 	var results []Menu
// 	err := json.Unmarshal([]byte(txt), &results)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	var licenseKeys = map[string][]string{
// 		"loganalyzer":               []string{"syslog"},
// 		"mc.Todolists.Index":        []string{"base"},
// 		"nm.auto.inspection":        []string{"tour_inspection"},
// 		"nm.data_flow.biz_analysis": []string{"nflow"},
// 		"nm.data_flow.live_view":    []string{"nflow"},
// 		"nm.ip_address.comparison":  []string{"ip-mgr"},
// 		"nm.ip_address.list":        []string{"ip-mgr"},
// 		"nm.ip_address.setting":     []string{"ip-mgr"},
// 		"nm.ip_address.trace_log":   []string{"ip-mgr"},
// 		"nm.monitor.topology":       []string{"base"},
// 		// "nm.resource1":                []string{"base"},
// 		//"nm.storage":                  []string{"storage"},
// 		//"nm.system2":                  []string{"base"},
// 		// "nm.virtualization":           []string{"vm"},
// 		"nm.vm_storage.object_browse": []string{"vm"},

// 		"app.products":         []string{"base", "nm", "storage", "vm"},
// 		"nm.home":              []string{"base", "nm", "storage", "vm"},
// 		"nm.monitor1":          []string{"base", "nm", "storage", "vm"},
// 		"nm.auto.operation1":   []string{"nm"},
// 		"nm.net_analysis1":     []string{"nm", "storage", "vm"},
// 		"nm.data_flow1":        []string{"base", "nm", "storage", "vm"},
// 		"nm.vm_storage1":       []string{"storage", "vm"},
// 		"nm.virtualization":    []string{"vm"},
// 		"nm.storage":           []string{"storage"},
// 		"nm.common_tools1":     []string{"base", "nm", "storage", "vm"},
// 		"nm.alarm_log1":        []string{"base", "nm", "storage", "vm"},
// 		"nm.history_report1\"": []string{"base", "nm", "storage", "vm"},
// 		"nm.resource1":         []string{"base", "nm", "storage", "vm"},
// 		"nm.system2":           []string{"base", "nm", "storage", "vm"},
// 	}

// 	weaver := &menuWeaver{hasLicense: func(ctx string, menu Menu) (bool, error) {

// 		perms := strings.Split(menu.License, ",")
// 		perms = append(perms, menu.UID)

// 		modules := []string{"base"}

// 		uid := menu.UID
// 		if strings.HasPrefix(menu.UID, "product-") {
// 			uid = strings.TrimPrefix(menu.UID, "product-")
// 			if uid == "wserver" {
// 				uid = "nm"
// 			}
// 		}

// 		if keys, ok := licenseKeys[uid]; ok && len(keys) > 0 {
// 			perms = append(perms, keys...)
// 		}

// 		for _, module := range modules {
// 			if module == "" {
// 				continue
// 			}

// 			if module == uid {
// 				return true, nil
// 			}
// 			if module == ctx {
// 				return true, nil
// 			}

// 			foundIdx := -1
// 			for i := range perms {
// 				if perms[i] == module {
// 					foundIdx = i
// 					break
// 				}
// 			}

// 			if foundIdx >= 0 {
// 				return true, nil
// 			}
// 		}

// 		return false, nil
// 	}}

// 	a, _ := weaver.deleteByLicense("ctx", results)
// 	bs, _ := json.MarshalIndent(a, "", "  ")
// 	t.Log(string(bs))
// }
