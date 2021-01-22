package main

import (

	// Standard packages
	"fmt"
	"errors"

	// Custom packages
	"temporal"
	"maketest"
	"types"
)

// Attempts to return a set of utilisations that sum to a total.
// Also attempts to ensure no single fraction is less than the given min value
func get_utilisation (min, total float64, attempts, fragments int) ([]float64, error) {
	for i := 0; i < attempts; i++ {
		ok := true
		us := temporal.Uunifast(total, fragments)
		for _, u := range us {
			if u < min {
				ok = false
				break
			}
		}
		if ok {
			return us, nil
		}
	}
	return []float64{}, errors.New("Unable to derive suitable utilisation values")
}

func main () {

	// Path at which to place test
	var path string = "/home/micrified/Thesis/tests"

	// The test environment
	var environment maketest.Environment = maketest.Environment{
		Generate_directory:    "/home/micrified/Go/src/rosgraph",
		Workspace_directory:   "/home/micrified/Thesis/workspace",
		ROS_directory:         "/home/micrified/Thesis/ros2_foxy",
		Analysis_directory:    "/home/micrified/Go/src/postprocess",
		Results_directory:     "/home/micrified/Thesis/results",
		Logfile_directory:     "/var/log",
		Logfile_name:          "ros.log",
	}

	// Desired total utilisation of the system under test
	var u_total float64    = 0.6

	// Minimum acceptable fraction utilisation of total
	var min_u float64      = 0.05

	// Maximum number of times to re-sample utilisation if unsuitable
	var max_u_attempts int = 5000

	// Number of chains involved in the test
	var n_chains int       = 5

	// Number of trials to run of the test
	start_n_trials, end_n_trials := 0, 50

	// Minimum, maximum, and step time (in microseconds) for a period of a chain
	min_t_us, max_t_us, step_us := 1000.0, 1000000.0, 50000.0

	// Starting and ending chain length
	start_chain_length, end_chain_length := 2, 10

	// Starting and ending executor count
	start_executor_count, end_executor_count := 4, 4

	// Whether to post-process
	var should_postprocess bool = true

	// Whether to reset the logfile before running a test
	var should_reset_logging bool = true

	// Whether to use custom timing
	var should_use_custom_timing = false

	for trial := start_n_trials; trial < end_n_trials; trial++ {

		// Obtain a base utilisation
		base_us, err := get_utilisation(min_u, u_total, max_u_attempts, n_chains)
		if nil != err {
			panic(err)
		}

		// Convert into (T,C) for each chain
		base_ts, err := temporal.Make_Temporal_Data(
			temporal.Range{Min: min_t_us, Max: max_t_us}, step_us, base_us)
		if nil != err {
			panic(err)
		}

		// Copy the base timing into a working timing set
		work_ts := make([]temporal.Temporal, len(base_ts))
		for i, t := range base_ts {
			work_ts[i] = temporal.Temporal{T: t.T, C: t.C}
			//work_ts[i] = temporal.Temporal{T: 1000000, C: 1000000}
		}

		for length := start_chain_length; length <= end_chain_length; length++ {

			for count := start_executor_count; count <= end_executor_count; count++ {

				// Create the names for both the PPE and normal version
				name_ppe := fmt.Sprintf("test_c%d_e%d_t%d_ppe", length, count, trial)
				name_std := fmt.Sprintf("test_c%d_e%d_t%d_std", length, count, trial)

				// Create rules
				ros_gen_rules := types.Rules{
					Name:                   name_ppe,
					Directory:              "/home/micrified/Go/src/rosgraph",
					Chain_count:            n_chains,
					Chain_avg_len:          length,
					Chain_merge_p:          0.0,
					Chain_sync_p:           0.0,
					Chain_variance:         0.25,
					Util_total:             u_total,
					Min_period_us:          int(min_t_us),
					Max_period_us:          int(max_t_us),
					Period_step_us:         step_us,
					Hyperperiod_count:      0,
					Max_duration_us:        60000000,
					PPE:                    true,
					Executor_count:         count,
					Random_seed:            trial,
					Logging_mode:           2,
				}

				// Test the PPE version
				fmt.Printf("Generating tests %s, and %s\n", name_ppe, name_std)
				err = maketest.Maketest(name_ppe, path, ros_gen_rules, should_use_custom_timing, should_postprocess, should_reset_logging,
					work_ts, environment)
				if nil != err {
					panic(err)
				}
				// Test the STD version
				ros_gen_rules.Name = name_std
				ros_gen_rules.PPE  = false

				// TODO: execute test
				err = maketest.Maketest(name_std, path, ros_gen_rules, should_use_custom_timing, should_postprocess, should_reset_logging,
					work_ts, environment)
				if nil != err {
					panic(err)
				}
			}

			// Add another chunk to the computation time of each chain.
			// + Recompute period
			// for i := 0; i < len(work_ts); i++ {
			// 	work_ts[i].C += base_ts[i].C
			// 	work_ts[i].T = work_ts[i].C / base_us[i]
			// }
		}
	}
}