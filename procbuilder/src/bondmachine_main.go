package main

import (
	"bondmachine"
	"encoding/binary"
	"encoding/json"
	"errors"
	"etherbond"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"procbuilder"
	"simbox"
	"sort"
	"strconv"
	"strings"
	"time"
	"udpbond"
)

type string_slice []string

func (i *string_slice) String() string {
	return fmt.Sprint(*i)
}

func (i *string_slice) Set(value string) error {
	for _, dt := range strings.Split(value, ",") {
		*i = append(*i, dt)
	}
	return nil
}

var debug = flag.Bool("d", false, "Debug")
var verbose = flag.Bool("v", false, "Verbose")
var commentedverilog = flag.Bool("comment-verilog", false, "Comment generated verilog")

var register_size = flag.Int("register-size", 8, "Number of bits per register (n-bit)")

var bondmachine_file = flag.String("bondmachine-file", "", "Filename of the bondmachine")

// Verilog processing
var create_verilog = flag.Bool("create-verilog", false, "Create default verilog files")
var verilog_flavor = flag.String("verilog-flavor", "iverilog", "Choose the type of verilog device. currently supported: iverilog,de10nano.")
var verilog_mapfile = flag.String("verilog-mapfile", "", "File mapping the device IO to bondmachine IO")
var verilog_simulation = flag.Bool("verilog-simulation", false, "Create simulation oriented verilog as default.")

var show_program_alias = flag.Bool("show-program-alias", false, "Show program alias for the processor")

// Domains processing
var list_domains = flag.Bool("list-domains", false, "Domain list")
var add_domains string_slice
var del_domains string_slice

// Processors
var list_processors = flag.Bool("list-processors", false, "Processor list")
var add_processor = flag.Int("add-processor", -1, "Add a processor of the given domain")

// TODO del-processor

// Inputs
var list_inputs = flag.Bool("list-inputs", false, "Inputs list")
var add_inputs = flag.Int("add-inputs", 0, "Inputs to add") // When adding we need how many new inputs, when removing we need which (thats why the list)
var del_inputs string_slice

//Outputs
var list_outputs = flag.Bool("list-outputs", false, "Outputs list")
var add_outputs = flag.Int("add-outputs", 0, "Outputs to add")
var del_outputs string_slice

var list_bonds = flag.Bool("list-bonds", false, "Bonds list")
var add_bond string_slice
var del_bonds string_slice

// TODO Shared objects
var list_shared_objects = flag.Bool("list-shared-objects", false, "Shared object list")
var add_shared_objects string_slice
var del_shared_objects string_slice
var list_processor_shared_object_links = flag.Bool("list-processor-shared-object-links", false, "Processor shared object link list")
var connect_processor_shared_object string_slice
var disconnect_processor_shared_object string_slice

var list_internal_inputs = flag.Bool("list-internal-inputs", false, "Internal inputs list")
var list_internal_outputs = flag.Bool("list-internal-outputs", false, "Internal outputs list")

// Dot output
var emit_dot = flag.Bool("emit-dot", false, "Emit dot file on stdout")
var dot_detail = flag.Int("dot-detail", 1, "Detail of infos on dot file 1-5")

// Assembly output
var show_program_disassembled = flag.Bool("show-program-disassembled", false, "Show disassebled program")
var multi_abstract_assembly_file = flag.String("multi-abstract-assembly-file", "", "Save the bondmachine as multi abstract assembly file")

var simbox_file = flag.String("simbox-file", "", "Filename of the simulation data file")

var sim = flag.Bool("sim", false, "Simulate bond machine")
var sim_interactions = flag.Int("sim-interactions", 10, "Simulation interaction")

var emu = flag.Bool("emu", false, "Emulate bond machine")
var emu_interactions = flag.Int("emu-interactions", 10, "Emulation interaction (0 means forever)")

var cluster_spec = flag.String("cluster-spec", "", "Etherbond or udpbond cluster Spec File ")
var peer_id = flag.Int("peer-id", -1, "Etherbond or udpbond Peer ID")

var use_etherbond = flag.Bool("use-etherbond", false, "Build including etherbond support")
var etherbond_flavor = flag.String("etherbond-flavor", "enc60j28", "Choose the type of ethernet device. currently supported: enc60j28.")
var etherbond_mapfile = flag.String("etherbond-mapfile", "", "File mapping the bondmachine IO the etherbond.")
var etherbond_macfile = flag.String("etherbond-macfile", "", "File mapping the bondmachine peers to MAC addresses.")

var use_udpbond = flag.Bool("use-udpbond", false, "Build including udpbond support")
var udpbond_flavor = flag.String("udpbond-flavor", "esp8266", "Choose the type of network device. currently supported: esp8266.")
var udpbond_mapfile = flag.String("udpbond-mapfile", "", "File mapping the bondmachine IO the udpbond.")
var udpbond_ipfile = flag.String("udpbond-ipfile", "", "File mapping the bondmachine peers to IP addresses.")
var udpbond_netconfig = flag.String("udpbond-netconfig", "", "JSON file containing the network configuration for udpbond")

var board_slow = flag.Bool("board-slow", false, "Board slow support")
var board_slow_factor = flag.Int("board-slow-factor", 1, "Board slow factor")

var basys3_7segment = flag.Bool("basys3-7segment", false, "Basys3 7 segments display support")
var basys3_7segment_map = flag.String("basys3-7segment-map", "", "Basys3 7 segments display mappings")

var attach_benchmark_core string_slice

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {
	rand.Seed(int64(time.Now().Unix()))

	flag.Var(&add_domains, "add-domains", "Comma-separated list of JSON machine files to add")
	flag.Var(&del_domains, "del-domains", "Comma-separated list of domain ID to delete")
	flag.Var(&del_inputs, "del-inputs", "Comma-separated list of input ID to delete")
	flag.Var(&del_outputs, "del-outputs", "Comma-separated list of output ID to delete")
	flag.Var(&del_bonds, "del-bonds", "Comma-separated list of bond ID to delete")
	flag.Var(&add_bond, "add-bond", "Bond with comma-separated endpoints")
	flag.Var(&add_shared_objects, "add-shared-objects", "Add a shared object")
	flag.Var(&del_shared_objects, "del-shared-objects", "Delete a shared object")
	flag.Var(&connect_processor_shared_object, "connect-processor-shared-object", "Connect a processor to a shared object")
	flag.Var(&disconnect_processor_shared_object, "disconnect-processor-shared-object", "Disconnect a processor from a shared object")
	flag.Var(&attach_benchmark_core, "attach-benchmark-core", "Attach a benchmark core")

	flag.Parse()
}
func lastAddr(n *net.IPNet) (net.IP, error) { // works when the n is a prefix, otherwise...
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("does not support IPv6 addresses.")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}
func main() {
	conf := new(bondmachine.Config)
	conf.Debug = *debug
	conf.Dotdetail = uint8(*dot_detail)
	conf.Commented_verilog = *commentedverilog

	var bmach *bondmachine.Bondmachine

	if *bondmachine_file != "" {
		if _, err := os.Stat(*bondmachine_file); err == nil {
			// Open the bondmachine file is exists
			if bondmachine_json, err := ioutil.ReadFile(*bondmachine_file); err == nil {
				var bmachj bondmachine.Bondmachine_json
				if err := json.Unmarshal([]byte(bondmachine_json), &bmachj); err == nil {
					bmach = (&bmachj).Dejsoner()
				} else {
					panic(err)
				}
			} else {
				panic(err)
			}
		} else {
			// Or create a new one
			bmach = new(bondmachine.Bondmachine)
			bmach.Rsize = uint8(*register_size)
		}

		bmach.Init()

		if &attach_benchmark_core != nil && len(attach_benchmark_core) == 2 {
			err := bmach.Attach_benchmark_core(attach_benchmark_core)
			check(err)
		}

		// Eventually create verilog files
		if *create_verilog {
			iomap := new(bondmachine.IOmap)
			if *verilog_mapfile != "" {
				if mapfile_json, err := ioutil.ReadFile(*verilog_mapfile); err == nil {
					if err := json.Unmarshal([]byte(mapfile_json), iomap); err != nil {
						panic(err)
					}
				} else {
					panic(err)
				}

			}
			//fmt.Println(iomap)

			// Precess the possible extra modules
			extramodules := make([]bondmachine.ExtraModule, 0)

			// Slower
			if *board_slow {
				em := new(bondmachine.Slow_extra)
				em.Slow_factor = strconv.Itoa(*board_slow_factor)

				if err := em.Check(bmach); err != nil {
					panic(err)
				}
				extramodules = append(extramodules, em)
			}

			// Etherbond
			//TODO
			if *use_etherbond {
				ethb := new(bondmachine.Etherbond_extra)

				config := new(etherbond.Config)
				config.Rsize = uint8(*register_size)

				ethb.Config = config
				ethb.Flavor = *etherbond_flavor

				if *cluster_spec != "" {
					if cluster, err := etherbond.UnmarshallCluster(config, *cluster_spec); err != nil {
						panic(err)
					} else {
						ethb.Cluster = cluster
					}
				} else {
					panic("A Cluster spec file is needed")
				}

				ethiomap := new(bondmachine.IOmap)
				if *etherbond_mapfile != "" {
					if mapfile_json, err := ioutil.ReadFile(*etherbond_mapfile); err == nil {
						if err := json.Unmarshal([]byte(mapfile_json), ethiomap); err != nil {
							panic(err)
						}
					} else {
						panic(err)
					}

				} else {
					panic(errors.New("Etherbond Mapfile needed"))
				}

				macmap := new(etherbond.Macs)
				if *etherbond_macfile != "" {
					if macfile_json, err := ioutil.ReadFile(*etherbond_macfile); err == nil {
						if err := json.Unmarshal([]byte(macfile_json), macmap); err != nil {
							panic(err)
						}
					} else {
						panic(err)
					}
				}

				ethb.Macs = macmap
				ethb.Maps = ethiomap
				ethb.PeerID = uint32(*peer_id)
				ethb.Mac = "0288" + fmt.Sprintf("%08d", *peer_id)

				if err := ethb.Check(bmach); err != nil {
					panic(err)
				}
				extramodules = append(extramodules, ethb)
			}
			if *use_udpbond {
				udpb := new(bondmachine.Udpbond_extra)

				// TODO Import the wiki configuration from file

				config := new(udpbond.Config)
				config.Rsize = uint8(*register_size)

				udpb.Config = config
				udpb.Flavor = *udpbond_flavor

				if *cluster_spec != "" {
					if cluster, err := udpbond.UnmarshallCluster(config, *cluster_spec); err != nil {
						panic(err)
					} else {
						udpb.Cluster = cluster
					}
				} else {
					panic("A Cluster spec file is needed")
				}

				ethiomap := new(bondmachine.IOmap)
				if *udpbond_mapfile != "" {
					if mapfile_json, err := ioutil.ReadFile(*udpbond_mapfile); err == nil {
						if err := json.Unmarshal([]byte(mapfile_json), ethiomap); err != nil {
							panic(err)
						}
					} else {
						panic(err)
					}

				} else {
					panic(errors.New("Udpbond Mapfile needed"))
				}

				macmap := new(udpbond.Ips)
				if *udpbond_ipfile != "" {
					if macfile_json, err := ioutil.ReadFile(*udpbond_ipfile); err == nil {
						if err := json.Unmarshal([]byte(macfile_json), macmap); err != nil {
							panic(err)
						}
					} else {
						panic(err)
					}
				}

				netparams := new(bondmachine.NetParameters)
				if *udpbond_netconfig != "" {
					if netconfig_json, err := ioutil.ReadFile(*udpbond_netconfig); err == nil {
						if err := json.Unmarshal([]byte(netconfig_json), netparams); err != nil {
							panic(err)
						}
					} else {
						panic(err)
					}
				}
				fmt.Println(netparams)
				udpb.NetParams = netparams
				udpb.Ips = macmap
				udpb.Maps = ethiomap
				udpb.PeerID = uint32(*peer_id)
				if ipst, ok := macmap.Assoc["peer_"+strconv.Itoa(*peer_id)]; ok {
					ip := strings.Split(ipst, "/")[0]
					port := strings.Split(ipst, ":")[1]
					udpb.Ip = ip
					udpb.Port = port
					_, nip, _ := net.ParseCIDR(strings.Split(ipst, ":")[0])
					brd, _ := lastAddr(nip)
					udpb.Broadcast = brd.String()
					//udpb.Netmask = nip.String()
				} else {
					panic(errors.New("Wrong IP"))
				}
				if err := udpb.Check(bmach); err != nil {
					panic(err)
				}
				extramodules = append(extramodules, udpb)
			}
			if *basys3_7segment {
				b37s := new(bondmachine.B37s)
				b37s.Mapped_output = *basys3_7segment_map
				extramodules = append(extramodules, b37s)
			}

			var flavor string

			if *verilog_simulation {
				flavor = *verilog_flavor + "_simulation"
			} else {
				flavor = *verilog_flavor
			}

			var sbox *simbox.Simbox

			if *verilog_simulation {
				if *simbox_file != "" {
					sbox = new(simbox.Simbox)
					if _, err := os.Stat(*simbox_file); err == nil {
						// Open the simbox file is exists
						if simbox_json, err := ioutil.ReadFile(*simbox_file); err == nil {
							if err := json.Unmarshal([]byte(simbox_json), sbox); err != nil {
								panic(err)
							}
						} else {
							panic(err)
						}
					}

				}
			}

			bmach.Write_verilog(conf, flavor, iomap, extramodules, sbox)
		}

		// All the operation are exclusive
		if *list_domains {
			fmt.Println(bmach.List_domains())
		} else if &add_domains != nil && len(add_domains) != 0 {
			for _, load_machine := range add_domains {
				if _, err := os.Stat(load_machine); err == nil {
					if jsonfile, err := ioutil.ReadFile(load_machine); err == nil {
						var machj procbuilder.Machine_json
						if err := json.Unmarshal([]byte(jsonfile), &machj); err == nil {
							mymachine := (&machj).Dejsoner()
							bmach.Domains = append(bmach.Domains, mymachine)
						} else {
							panic(err)
						}
					} else {
						panic(err)
					}
				} else {
					fmt.Println(load_machine + " file not found, ignoring it.")
				}
			}
		} else if (&del_domains != nil) && len(del_domains) != 0 {
			for _, remove_domain := range del_domains {
				if remove_domain_id, err := strconv.Atoi(remove_domain); err == nil {
					if remove_domain_id < len(bmach.Domains)-1 {
						bmach.Domains = append(bmach.Domains[:remove_domain_id], bmach.Domains[remove_domain_id+1:]...)
					} else if remove_domain_id == len(bmach.Domains)-1 {
						bmach.Domains = bmach.Domains[:remove_domain_id]
					} else {
						fmt.Println(remove_domain + " not a valid domain id, ignoring it.")
					}
				} else {
					fmt.Println(remove_domain + " not a valid domain id, ignoring it.")
				}
			}
			// TODO Include the check of unbounded processors

		} else if *list_inputs {
			for i, inp := range bmach.List_inputs() {
				fmt.Printf("%d %s\n", i, inp)
			}
		} else if *add_inputs != 0 {
			for i := 0; i < *add_inputs; i++ {
				message, err := bmach.Add_input()
				check(err)
				if *debug {
					log.Println(message)
				} else if *verbose {
					fmt.Println(message)
				}
			}
		} else if (&del_inputs != nil) && len(del_inputs) != 0 {
			// Reorder the inputs to delete, last first
			todelete := make([]int, 0)
			for _, inp := range del_inputs {
				if value, ok := strconv.Atoi(inp); ok == nil {
					pcheck := false
					for _, i := range todelete {
						if i == value {
							pcheck = true
							break
						}
					}
					if !pcheck && value < bmach.Inputs {
						todelete = append(todelete, value)
					}
				}
			}
			sort.Ints(todelete)
			for i, _ := range todelete {
				// Remove the inputs, higher first
				bmach.Del_input(todelete[len(todelete)-i-1])
			}
		} else if *list_outputs {
			for i, outp := range bmach.List_outputs() {
				fmt.Printf("%d %s\n", i, outp)
			}
		} else if *add_outputs != 0 {
			for i := 0; i < *add_outputs; i++ {
				message, err := bmach.Add_output()
				check(err)
				if *debug {
					log.Println(message)
				} else if *verbose {
					fmt.Println(message)
				}
			}
		} else if (&del_outputs != nil) && len(del_outputs) != 0 {
			// Reorder the outputs to delete, last first
			todelete := make([]int, 0)
			for _, inp := range del_outputs {
				if value, ok := strconv.Atoi(inp); ok == nil {
					pcheck := false
					for _, i := range todelete {
						if i == value {
							pcheck = true
							break
						}
					}
					if !pcheck && value < bmach.Outputs {
						todelete = append(todelete, value)
					}
				}
			}
			sort.Ints(todelete)
			for i, _ := range todelete {
				// Remove the outputs, higher first
				bmach.Del_output(todelete[len(todelete)-i-1])
			}
		} else if *list_processors {
			fmt.Print(bmach.List_processors())
		} else if *add_processor != -1 {
			message, err := bmach.Add_processor(*add_processor)
			check(err)
			fmt.Println(message)
		} else if *list_bonds {
			for i, bond := range bmach.List_bonds() {
				fmt.Printf("%d %s\n", i, bond)
			}
		} else if *list_internal_inputs {
			for _, inp := range bmach.List_internal_inputs() {
				fmt.Println(inp)
			}
		} else if *list_internal_outputs {
			for _, outp := range bmach.List_internal_outputs() {
				fmt.Println(outp)
			}
		} else if *emit_dot && !*sim {
			fmt.Print(bmach.Dot(conf, "", nil, nil))
		} else if *show_program_disassembled {
			// TODO Finish
		} else if *multi_abstract_assembly_file != "" {
			// TODO Temporary, clean up code!
			mu, _ := bmach.GetMultiAssembly()
			// Write the multi_abstract_assembly_file file
			mufile, err := os.Create(*multi_abstract_assembly_file)
			check(err)
			defer mufile.Close()
			b, errj := json.Marshal(mu)
			check(errj)
			_, err = mufile.WriteString(string(b))
			check(err)
		} else if *show_program_alias {
			pa, _ := bmach.GetProgramsAlias()
			for i, al := range pa {
				alfile, err := os.Create("p" + strconv.Itoa(i) + ".alias")
				check(err)
				defer alfile.Close()
				_, err = alfile.WriteString(string(al))
				check(err)
			}
		} else if &add_bond != nil && len(add_bond) == 2 {
			bmach.Add_bond(add_bond)
		} else if (&del_bonds != nil) && len(del_bonds) != 0 {
			for _, remove_bond := range del_bonds {
				if remove_bond_id, err := strconv.Atoi(remove_bond); err == nil {
					if remove_bond_id < len(bmach.Links) {
						bmach.Del_bond(remove_bond_id)
					} else {
						fmt.Println(remove_bond + " not a valid bond id, ignoring it.")
					}
				} else {
					fmt.Println(remove_bond + " not a valid bond id, ignoring it.")
				}
			}
		} else if *list_shared_objects {
			fmt.Print(bmach.List_shared_objects())
		} else if &add_shared_objects != nil && len(add_shared_objects) > 0 {
			bmach.Add_shared_objects(add_shared_objects)
		} else if *list_processor_shared_object_links {
			fmt.Print(bmach.List_processor_shared_object_links())
		} else if &connect_processor_shared_object != nil && len(connect_processor_shared_object) == 2 {
			bmach.Connect_processor_shared_object(connect_processor_shared_object)
		} else if *sim {
			var sbox *simbox.Simbox
			if *simbox_file != "" {
				sbox = new(simbox.Simbox)
				if _, err := os.Stat(*simbox_file); err == nil {
					// Open the simbox file is exists
					if simbox_json, err := ioutil.ReadFile(*simbox_file); err == nil {
						if err := json.Unmarshal([]byte(simbox_json), sbox); err != nil {
							panic(err)
						}
					} else {
						panic(err)
					}
				}

			}

			vm := new(bondmachine.VM)
			vm.Bmach = bmach
			err := vm.Init()
			check(err)

			var pstatevm *bondmachine.VM

			// Build the simulation configuration
			sconfig := new(bondmachine.Sim_config)
			scerr := sconfig.Init(sbox, vm, conf)
			check(scerr)

			// Build the simulation driver
			sdrive := new(bondmachine.Sim_drive)
			sderr := sdrive.Init(sbox, vm)
			check(sderr)

			// Build the simultion report
			srep := new(bondmachine.Sim_report)
			srerr := srep.Init(sbox, vm)
			check(srerr)

			lerr := vm.Launch_processors(sbox)
			check(lerr)

			var intlen_s string

			if *emit_dot {
				pstatevm = new(bondmachine.VM)
				pstatevm.Bmach = bmach
				err := pstatevm.Init()
				check(err)

				sim_int_s := strconv.Itoa(*sim_interactions)
				intlen := len(sim_int_s)
				intlen_s = strconv.Itoa(intlen)
			}

			for i := uint64(0); i < uint64(*sim_interactions); i++ {

				// This will get actions eventually to do on this tick
				if act, exist_actions := sdrive.AbsSet[i]; exist_actions {
					for k, val := range act {
						*sdrive.Injectables[k] = val
					}
				}

				// TODO Periodic set

				if *emit_dot {
					gvfile := bmach.Dot(conf, "", vm, pstatevm)
					filename := fmt.Sprintf("graphviz%0"+intlen_s+"d", int(i))
					f, err := os.Create(filename + ".dot")
					check(err)
					_, err = f.WriteString(gvfile)
					check(err)
					f.Close()

					pstatevm.CopyState(vm)
				}

				result, err := vm.Step(sconfig)
				check(err)

				fmt.Print(result)

				// This will get value to show on this tick
				if slist, exist_shows := srep.AbsShow[i]; exist_shows {
					for k, _ := range slist {
						fmt.Println(*srep.Showables[k])
					}
				}

				// This will get value to show on periodic ticks
				for j, slist := range srep.PerShow {
					if i%j == 0 {
						for k, _ := range slist {
							fmt.Print(*srep.Showables[k], " ")
						}
						fmt.Println("")
					}
				}

				// This will get value to report on this tick
				if rep, exist_reports := srep.AbsGet[i]; exist_reports {
					for k, _ := range rep {
						rep[k] = *srep.Reportables[k]
					}
					fmt.Println("TEMPORARY:", rep)
				}

				// TODO Periodic get
				// TODO Write to a yet to be created report data structure

			}
		} else if *emu {
			vm := new(bondmachine.VM)
			vm.Bmach = bmach
			err := vm.Init()
			check(err)

			// the emulation configuration is not really needed
			sconfig := new(bondmachine.Sim_config)
			scerr := sconfig.Init(nil, vm, conf)
			check(scerr)

			lerr := vm.Launch_processors(nil)
			check(lerr)

			for i := uint64(0); ; {
				if *emu_interactions != 0 {
					if i >= uint64(*emu_interactions) {
						break
					}
				}

				_, err := vm.Step(sconfig)
				check(err)

				if *emu_interactions != 0 {
					i++
				}

			}
		}

		// Write the bondmachine file
		f, err := os.Create(*bondmachine_file)
		check(err)
		defer f.Close()
		b, errj := json.Marshal(bmach.Jsoner())
		check(errj)
		_, err = f.WriteString(string(b))
		check(err)
	} else {
		panic("bondmachine-file is a mandatory option")
	}
}
