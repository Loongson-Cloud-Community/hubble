// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Hubble

package observe

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	flowpb "github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/pkg/identity"
	monitorAPI "github.com/cilium/cilium/pkg/monitor/api"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoBlacklist(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{
		"--from-ip", "1.2.3.4",
	}))
	assert.Nil(t, f.blacklist, "blacklist should be nil")
}

// The default filter should be empty.
func TestDefaultFilter(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{}))
	assert.Nil(t, f.whitelist)
	assert.Nil(t, f.blacklist)
}

func TestConflicts(t *testing.T) {

	tcs := map[string]struct {
		flags    []string
		conflict bool
		msg      string
	}{
		"ip fqdn conflict": {
			flags: []string{
				"--from-ip", "1.2.3.4",
				"--from-fqdn", "doesnt.work",
			},
			conflict: true,
			msg:      "filters --from-fqdn and --from-ip cannot be combined",
		},
		"svc from-svc conflict": {
			flags: []string{
				"--from-service", "foo",
				"--service", "bar",
			},
			conflict: true,
			msg:      "filters --service and --from-service cannot be combined",
		},
		"from-svc to-svc ok": {
			flags: []string{
				"--to-service", "foo",
				"--from-service", "bar",
			},
		},
		"complex ok": {
			flags: []string{
				"--to-service", "foo",
				"--from-service", "bar",
				"--from-identity", "world",
				"--to-identity", "world",
				"--to-ip", "1.2.3.4",
				"--from-pod", "blib",
				"--from-port", "1172",
				"--to-port", "11245",
				"--workload", "buzz",
			},
		},
		"complex fqdn ok": {
			flags: []string{
				"--to-service", "foo",
				"--from-service", "bar",
				"--from-identity", "world",
				"--to-identity", "world",
				"--to-ip", "1.2.3.4",
				"--from-fqdn", "blib.example.com",
				"--from-port", "1172",
				"--to-port", "11245",
				"--workload", "buzz",
			},
		},
		"complex with namespace ok": {
			flags: []string{
				"--to-service", "foo",
				"--from-service", "bar",
				"--from-identity", "world",
				"--to-identity", "world",
				"--to-fqdn", "example.com",
				"--from-pod", "blib",
				"--from-namespace", "test",
				"--from-port", "1172",
				"--to-port", "11245",
				"--workload", "buzz",
			},
		},
		"complex with namespace and service ok": {
			flags: []string{
				"--service", "bar",
				"--from-identity", "world",
				"--to-identity", "world",
				"--pod", "blib",
				"--namespace", "test",
				"--from-port", "1172",
				"--to-port", "11245",
				"--workload", "buzz",
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			err := cmd.Flags().Parse(tc.flags)
			if tc.conflict {
				require.Error(t, err)
				if tc.msg != "" {
					assert.Contains(t,
						err.Error(),
						tc.msg,
					)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConflicts_Table(t *testing.T) {

	conflicts := [][]string{
		{"--port", "--to-port"},
		{"--port", "--from-port"},
		{"--identity", "--to-identity"},
		{"--identity", "--from-identity"},
		{"--workload", "--to-workload"},
		{"--workload", "--from-workload"},
		{"--fqdn", "--from-fqdn"},
		{"--fqdn", "--to-fqdn"},
		{"--ip", "--from-ip"},
		{"--ip", "--to-ip"},
		{"--label", "--from-label"},
		{"--label", "--to-label"},
		{"--service", "--from-service"},
		{"--service", "--to-service"},
		{"--pod", "--from-pod"},
		{"--pod", "--to-pod"},
		{"--namespace", "--from-namespace"},
		{"--namespace", "--to-namespace"},

		{"--ip", "--fqdn"},
		{"--ip", "--from-fqdn"},
		{"--ip", "--to-fqdn"},
		{"--from-ip", "--from-fqdn"},
		{"--from-ip", "--fqdn"},
		{"--to-ip", "--fqdn"},
		{"--to-ip", "--to-fqdn"},
		{"--ip", "--pod"},
		{"--ip", "--from-pod"},
		{"--ip", "--to-pod"},
		{"--from-ip", "--from-pod"},
		{"--from-ip", "--pod"},
		{"--to-ip", "--pod"},
		{"--to-ip", "--to-pod"},
		{"--ip", "--namespace"},
		{"--ip", "--from-namespace"},
		{"--ip", "--to-namespace"},
		{"--from-ip", "--from-namespace"},
		{"--from-ip", "--namespace"},
		{"--to-ip", "--namespace"},
		{"--to-ip", "--to-namespace"},

		{"--pod", "--fqdn"},
		{"--pod", "--from-fqdn"},
		{"--pod", "--to-fqdn"},
		{"--from-pod", "--from-fqdn"},
		{"--from-pod", "--fqdn"},
		{"--from-pod", "--namepace"},
		{"--to-pod", "--fqdn"},
		{"--to-pod", "--to-fqdn"},
		{"--to-pod", "--namepace"},

		{"--namespace", "--fqdn"},
		{"--namespace", "--from-fqdn"},
		{"--namespace", "--to-fqdn"},
		{"--namespace", "--from-service"},
		{"--namespace", "--to-service"},
		{"--from-namespace", "--from-fqdn"},
		{"--from-namespace", "--fqdn"},
		{"--to-namespace", "--fqdn"},
		{"--to-namespace", "--to-fqdn"},
	}

	for _, con := range conflicts {
		t.Run(fmt.Sprintf("conflict %s - %s", con[0], con[1]), func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			require.Error(t, cmd.Flags().Parse([]string{con[0], "foo", con[1], "bar"}))
		})
	}
}

func TestTrailingNot(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	err := cmd.Flags().Parse([]string{
		"--from-ip", "1.2.3.4",
		"--not",
	})
	require.NoError(t, err)

	err = handleFlowArgs(os.Stdout, f, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trailing --not")
}

func TestFilterDispatch(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{
		"--from-ip", "1.2.3.4",
		"--from-ip", "5.6.7.8",
		"--not",
		"--to-ip", "5.5.5.5",
		"--verdict", "DROPPED",
		"-t", "l7", // int:129 in cilium-land
	}))

	require.NoError(t, handleFlowArgs(os.Stdout, f, false))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{
				SourceIp: []string{"1.2.3.4", "5.6.7.8"},
				Verdict:  []flowpb.Verdict{flowpb.Verdict_DROPPED},
				EventType: []*flowpb.EventTypeFilter{
					{Type: monitorAPI.MessageTypeAccessLog},
				},
			},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
		cmpopts.IgnoreUnexported(flowpb.EventTypeFilter{}),
	); diff != "" {
		t.Errorf("whitelist filter mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{
				DestinationIp: []string{"5.5.5.5"},
			},
		},
		f.blacklist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("blacklist filter mismatch (-want +got):\n%s", diff)
	}
}

func TestFilterLeftRight(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{
		"--ip", "1.2.3.4",
		"--ip", "5.6.7.8",
		"--verdict", "DROPPED",
		"--not", "--pod", "deathstar",
		"--not", "--http-status", "200",
		"--http-method", "get",
		"--http-path", "/page/\\d+",
		"--node-name", "k8s*",
	}))

	require.NoError(t, handleFlowArgs(os.Stdout, f, false))

	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{
				SourceIp:   []string{"1.2.3.4", "5.6.7.8"},
				Verdict:    []flowpb.Verdict{flowpb.Verdict_DROPPED},
				HttpMethod: []string{"get"},
				HttpPath:   []string{"/page/\\d+"},
				NodeName:   []string{"k8s*"},
			},
			{
				DestinationIp: []string{"1.2.3.4", "5.6.7.8"},
				Verdict:       []flowpb.Verdict{flowpb.Verdict_DROPPED},
				HttpMethod:    []string{"get"},
				HttpPath:      []string{"/page/\\d+"},
				NodeName:      []string{"k8s*"},
			},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("whitelist filter mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{
				SourcePod:      []string{"deathstar"},
				HttpStatusCode: []string{"200"},
			},
			{
				DestinationPod: []string{"deathstar"},
				HttpStatusCode: []string{"200"},
			},
		},
		f.blacklist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("blacklist filter mismatch (-want +got):\n%s", diff)
	}
}

func TestFilterType(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.Error(t, cmd.Flags().Parse([]string{
		"-t", "some-invalid-type",
	}))

	require.Error(t, cmd.Flags().Parse([]string{
		"-t", "trace:some-invalid-sub-type",
	}))

	require.Error(t, cmd.Flags().Parse([]string{
		"-t", "agent:Policy updated",
	}))

	require.NoError(t, cmd.Flags().Parse([]string{
		"-t", "254",
		"-t", "255:127",
		"-t", "trace:to-endpoint",
		"-t", "trace:from-endpoint",
		"-t", strconv.Itoa(monitorAPI.MessageTypeTrace) + ":" + strconv.Itoa(monitorAPI.TraceToHost),
		"-t", "agent",
		"-t", "agent:3",
		"-t", "agent:policy-updated",
		"-t", "agent:service-deleted",
	}))

	require.NoError(t, handleFlowArgs(os.Stdout, f, false))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{
				EventType: []*flowpb.EventTypeFilter{
					{
						Type: 254,
					},

					{
						Type:         255,
						MatchSubType: true,
						SubType:      127,
					},
					{
						Type:         monitorAPI.MessageTypeTrace,
						MatchSubType: true,
						SubType:      monitorAPI.TraceToLxc,
					},
					{
						Type:         monitorAPI.MessageTypeTrace,
						MatchSubType: true,
						SubType:      monitorAPI.TraceFromLxc,
					},
					{
						Type:         monitorAPI.MessageTypeTrace,
						MatchSubType: true,
						SubType:      monitorAPI.TraceToHost,
					},
					{
						Type: monitorAPI.MessageTypeAgent,
					},
					{
						Type:         monitorAPI.MessageTypeAgent,
						MatchSubType: true,
						SubType:      3,
					},
					{
						Type:         monitorAPI.MessageTypeAgent,
						MatchSubType: true,
						SubType:      int32(monitorAPI.AgentNotifyPolicyUpdated),
					},
					{
						Type:         monitorAPI.MessageTypeAgent,
						MatchSubType: true,
						SubType:      int32(monitorAPI.AgentNotifyServiceDeleted),
					},
				},
			},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
		cmpopts.IgnoreUnexported(flowpb.EventTypeFilter{}),
	); diff != "" {
		t.Errorf("filter mismatch (-want +got):\n%s", diff)
	}
}

func TestLabels(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	err := cmd.Flags().Parse([]string{
		"--label", "k1=v1,k2=v2",
		"-l", "k3",
	})
	require.NoError(t, err)
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{SourceLabel: []string{"k1=v1,k2=v2", "k3"}},
			{DestinationLabel: []string{"k1=v1,k2=v2", "k3"}},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	assert.Nil(t, f.blacklist)
}

func TestFromToWorkloadCombined(t *testing.T) {
	t.Run("single filter", func(t *testing.T) {
		f := newFlowFilter()
		cmd := newFlowsCmdWithFilter(viper.New(), f)

		require.NoError(t, cmd.Flags().Parse([]string{"--from-pod", "cilium", "--to-workload", "app"}))
		if diff := cmp.Diff(
			[]*flowpb.FlowFilter{
				{
					SourcePod:           []string{"cilium"},
					DestinationWorkload: []*flowpb.Workload{{Name: "app"}},
				},
			},
			f.whitelist.flowFilters(),
			cmpopts.IgnoreUnexported(flowpb.FlowFilter{}, flowpb.Workload{}),
		); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		assert.Nil(t, f.blacklist)
	})

	t.Run("two filters", func(t *testing.T) {
		f := newFlowFilter()
		cmd := newFlowsCmdWithFilter(viper.New(), f)

		require.NoError(t, cmd.Flags().Parse([]string{"--pod", "cilium", "--to-workload", "app"}))
		if diff := cmp.Diff(
			[]*flowpb.FlowFilter{
				{
					SourcePod:           []string{"cilium"},
					DestinationWorkload: []*flowpb.Workload{{Name: "app"}},
				},
				{
					DestinationPod:      []string{"cilium"},
					DestinationWorkload: []*flowpb.Workload{{Name: "app"}},
				},
			},
			f.whitelist.flowFilters(),
			cmpopts.IgnoreUnexported(flowpb.FlowFilter{}, flowpb.Workload{}),
		); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		assert.Nil(t, f.blacklist)
	})
}

func TestIdentity(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{"--identity", "1", "--identity", "2"}))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{SourceIdentity: []uint32{1, 2}},
			{DestinationIdentity: []uint32{1, 2}},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	assert.Nil(t, f.blacklist)

	// reserved identities
	for _, id := range identity.GetAllReservedIdentities() {
		t.Run(id.String(), func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			require.NoError(t, cmd.Flags().Parse([]string{"--identity", id.String()}))
			if diff := cmp.Diff(
				[]*flowpb.FlowFilter{
					{SourceIdentity: []uint32{id.Uint32()}},
					{DestinationIdentity: []uint32{id.Uint32()}},
				},
				f.whitelist.flowFilters(),
				cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			assert.Nil(t, f.blacklist)
		})
	}
}

func TestFromIdentity(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{"--from-identity", "1", "--from-identity", "2"}))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{SourceIdentity: []uint32{1, 2}},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	assert.Nil(t, f.blacklist)

	// reserved identities
	for _, id := range identity.GetAllReservedIdentities() {
		t.Run(id.String(), func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			require.NoError(t, cmd.Flags().Parse([]string{"--from-identity", id.String()}))
			if diff := cmp.Diff(
				[]*flowpb.FlowFilter{
					{SourceIdentity: []uint32{id.Uint32()}},
				},
				f.whitelist.flowFilters(),
				cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			assert.Nil(t, f.blacklist)
		})
	}
}

func TestToIdentity(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{"--to-identity", "1", "--to-identity", "2"}))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{DestinationIdentity: []uint32{1, 2}},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	assert.Nil(t, f.blacklist)

	// reserved identities
	for _, id := range identity.GetAllReservedIdentities() {
		t.Run(id.String(), func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			require.NoError(t, cmd.Flags().Parse([]string{"--to-identity", id.String()}))
			if diff := cmp.Diff(
				[]*flowpb.FlowFilter{
					{DestinationIdentity: []uint32{id.Uint32()}},
				},
				f.whitelist.flowFilters(),
				cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			assert.Nil(t, f.blacklist)
		})
	}
}

func TestFromToIdentityCombined(t *testing.T) {
	t.Run("single filter", func(t *testing.T) {
		f := newFlowFilter()
		cmd := newFlowsCmdWithFilter(viper.New(), f)

		require.NoError(t, cmd.Flags().Parse([]string{"--from-pod", "cilium", "--to-identity", "42"}))
		if diff := cmp.Diff(
			[]*flowpb.FlowFilter{
				{SourcePod: []string{"cilium"}, DestinationIdentity: []uint32{42}},
			},
			f.whitelist.flowFilters(),
			cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
		); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		assert.Nil(t, f.blacklist)
	})

	t.Run("two filters", func(t *testing.T) {
		f := newFlowFilter()
		cmd := newFlowsCmdWithFilter(viper.New(), f)

		require.NoError(t, cmd.Flags().Parse([]string{"--pod", "cilium", "--to-identity", "42"}))
		if diff := cmp.Diff(
			[]*flowpb.FlowFilter{
				{SourcePod: []string{"cilium"}, DestinationIdentity: []uint32{42}},
				{DestinationPod: []string{"cilium"}, DestinationIdentity: []uint32{42}},
			},
			f.whitelist.flowFilters(),
			cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
		); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
		assert.Nil(t, f.blacklist)
	})
}

func TestInvalidIdentity(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.Error(t, cmd.Flags().Parse([]string{"--from-identity", "bad"}))
	require.Error(t, cmd.Flags().Parse([]string{"--to-identity", "bad"}))
	require.Error(t, cmd.Flags().Parse([]string{"--identity", "bad"}))
}

func TestTcpFlags(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	// valid TCP flags
	validflags := []string{"SYN", "syn", "FIN", "RST", "PSH", "ACK", "URG", "ECE", "CWR", "NS", "syn,ack"}
	for _, f := range validflags {
		require.NoError(t, cmd.Flags().Parse([]string{"--tcp-flags", f}))                               // single --tcp-flags
		require.NoError(t, cmd.Flags().Parse([]string{"--tcp-flags", f, "--tcp-flags", "syn"}))         // multiple --tcp-flags
		require.NoError(t, cmd.Flags().Parse([]string{"--tcp-flags", f, "--not", "--tcp-flags", "NS"})) // --not --tcp-flags
	}

	// invalid TCP flags
	invalidflags := []string{"unknown", "syn,unknown", "unknown,syn", "syn,", ",syn"}
	for _, f := range invalidflags {
		require.Error(t, cmd.Flags().Parse([]string{"--tcp-flags", f}))
	}
}

func TestUuid(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{"--uuid", "b9fab269-04ae-495c-9d12-b6c36d41de0d"}))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{Uuid: []string{"b9fab269-04ae-495c-9d12-b6c36d41de0d"}},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	assert.Nil(t, f.blacklist)
}

func TestTrafficDirection(t *testing.T) {
	tt := []struct {
		name    string
		flags   []string
		filters []*flowpb.FlowFilter
		err     string
	}{
		{
			name:  "ingress",
			flags: []string{"--traffic-direction", "ingress"},
			filters: []*flowpb.FlowFilter{
				{TrafficDirection: []flowpb.TrafficDirection{flowpb.TrafficDirection_INGRESS}},
			},
		},
		{
			name:  "egress",
			flags: []string{"--traffic-direction", "egress"},
			filters: []*flowpb.FlowFilter{
				{TrafficDirection: []flowpb.TrafficDirection{flowpb.TrafficDirection_EGRESS}},
			},
		},
		{
			name:  "mixed case",
			flags: []string{"--traffic-direction", "INGRESS", "--traffic-direction", "EgrEss"},
			filters: []*flowpb.FlowFilter{
				{
					TrafficDirection: []flowpb.TrafficDirection{
						flowpb.TrafficDirection_INGRESS,
						flowpb.TrafficDirection_EGRESS,
					},
				},
			},
		},
		{
			name:  "invalid",
			flags: []string{"--traffic-direction", "to the moon"},
			err:   "to the moon: invalid traffic direction, expected ingress or egress",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			err := cmd.Flags().Parse(tc.flags)
			diff := cmp.Diff(tc.filters, f.whitelist.flowFilters(), cmpopts.IgnoreUnexported(flowpb.FlowFilter{}))
			if diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if tc.err != "" {
				require.Errorf(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			assert.Nil(t, f.blacklist)
		})
	}
}

func TestHTTPURL(t *testing.T) {
	f := newFlowFilter()
	cmd := newFlowsCmdWithFilter(viper.New(), f)

	require.NoError(t, cmd.Flags().Parse([]string{"--http-url", `http://.*cilium\.io/foo`, "--http-url", `http://www\.cilium\.io/bar`}))
	if diff := cmp.Diff(
		[]*flowpb.FlowFilter{
			{HttpUrl: []string{`http://.*cilium\.io/foo`, `http://www\.cilium\.io/bar`}},
		},
		f.whitelist.flowFilters(),
		cmpopts.IgnoreUnexported(flowpb.FlowFilter{}),
	); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	assert.Nil(t, f.blacklist)
}

func TestNamespace(t *testing.T) {

	tt := []struct {
		name    string
		flags   []string
		filters []*flowpb.FlowFilter
		err     string
	}{
		{
			name:  "Any request from or to namespace with port",
			flags: []string{"--namespace", "cilium", "--port", "443"},
			filters: []*flowpb.FlowFilter{
				{SourcePod: []string{"cilium/"}, SourcePort: []string{"443"}},
				{DestinationPod: []string{"cilium/"}, DestinationPort: []string{"443"}},
			},
		},
		{
			name:  "Any request from namespace with port",
			flags: []string{"--from-namespace", "cilium", "--from-port", "443"},
			filters: []*flowpb.FlowFilter{
				{SourcePod: []string{"cilium/"}, SourcePort: []string{"443"}},
			},
		},
		{
			name:  "Any request to namespace",
			flags: []string{"--to-namespace", "cilium"},
			filters: []*flowpb.FlowFilter{
				{DestinationPod: []string{"cilium/"}},
			},
		},
		{
			name:  "Any request to namespace with port",
			flags: []string{"--to-namespace", "cilium", "--port", "443"},
			filters: []*flowpb.FlowFilter{
				{DestinationPod: []string{"cilium/"}, SourcePort: []string{"443"}},
				{DestinationPod: []string{"cilium/"}, DestinationPort: []string{"443"}},
			},
		},
		{
			name:  "Any request from or to namespace and pod",
			flags: []string{"--namespace", "cilium", "--pod", "foo-9c76d6c95-tf788"},
			filters: []*flowpb.FlowFilter{
				{SourcePod: []string{"cilium/foo-9c76d6c95-tf788"}},
				{DestinationPod: []string{"cilium/foo-9c76d6c95-tf788"}},
			},
		},
		{
			name:  "Any request from or to namespace and svc",
			flags: []string{"--namespace", "cilium", "--service", "foo"},
			filters: []*flowpb.FlowFilter{
				{SourceService: []string{"cilium/foo"}},
				{DestinationService: []string{"cilium/foo"}},
			},
		},
		{
			name:  "Any request from or to one of namespaces and svcs",
			flags: []string{"--namespace", "cilium", "--namespace", "kube-system", "--service", "foo", "--service", "bar"},
			filters: []*flowpb.FlowFilter{
				{SourceService: []string{"cilium/foo", "kube-system/foo", "cilium/bar", "kube-system/bar"}},
				{DestinationService: []string{"cilium/foo", "kube-system/foo", "cilium/bar", "kube-system/bar"}},
			},
		},
		{
			name:  "Any request to namespace and pod with namespace",
			flags: []string{"--to-namespace", "cilium", "--to-pod", "cilium/foo-9c76d6c95-tf788"},
			filters: []*flowpb.FlowFilter{
				{DestinationPod: []string{"cilium/foo-9c76d6c95-tf788"}},
			},
		},
		{
			name:  "Any request from namespace to pod with namespace",
			flags: []string{"--from-namespace", "kube-system", "--to-pod", "cilium/foo-9c76d6c95-tf788"},
			filters: []*flowpb.FlowFilter{
				{SourcePod: []string{"kube-system/"}, DestinationPod: []string{"cilium/foo-9c76d6c95-tf788"}},
			},
		},
		{
			name:    "conflicting pod and namespace",
			flags:   []string{"--to-namespace", "kube-system", "--to-pod", "cilium/foo-9c76d6c95-tf788"},
			filters: []*flowpb.FlowFilter{},
			err:     `conflicting namepace: "kube-system" does not contain "cilium/foo-9c76d6c95-tf788"`,
		},
		{
			name:    "conflicting svc and namespace",
			flags:   []string{"--from-namespace", "kube-system", "--from-service", "cilium/foo"},
			filters: []*flowpb.FlowFilter{},
			err:     `conflicting namepace: "kube-system" does not contain "cilium/foo"`,
		},
		{
			name:    "conflicting svc and pod",
			flags:   []string{"--from-service", "cilium/foo", "--from-pod", "kube-system/hubble"},
			filters: []*flowpb.FlowFilter{},
			err:     `conflicting namepace: namespace of service "cilium/foo" conflict with pod "kube-system/hubble"`,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			f := newFlowFilter()
			cmd := newFlowsCmdWithFilter(viper.New(), f)
			err := cmd.Flags().Parse(tc.flags)
			if tc.err != "" {
				require.Errorf(t, err, tc.err)
				return
			} else {
				require.NoError(t, err)
			}
			assert.Nil(t, f.blacklist)
			diff := cmp.Diff(tc.filters, f.whitelist.flowFilters(), cmpopts.IgnoreUnexported(flowpb.FlowFilter{}))
			if diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
