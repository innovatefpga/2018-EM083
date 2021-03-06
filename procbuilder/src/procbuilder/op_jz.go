package procbuilder

// TODO This is the ROM, change it to halndle also the RAM case

import (
	"errors"
	"strconv"
	"strings"
)

// The Jz opcode is both a basic instruction and a template for other instructions.
type Jz struct{}

func (op Jz) Op_get_name() string {
	return "jz"
}

func (op Jz) Op_get_desc() string {
	return "Zero conditional jump"
}

func (op Jz) Op_show_assembler(arch *Arch) string {
	opbits := arch.Opcodes_bits()
	result := "jz [" + strconv.Itoa(int(arch.R)) + "(Reg)] [" + strconv.Itoa(int(arch.O)) + "(ROM Address)]	// Conditional jump [" + strconv.Itoa(opbits+int(arch.R)+int(arch.O)) + "]\n"
	return result
}

func (op Jz) Op_get_instruction_len(arch *Arch) int {
	opbits := arch.Opcodes_bits()
	return opbits + int(arch.R) + int(arch.O) // The bits for the opcode + bits for a register + bits for the rom address
}

func (op Jz) Op_instruction_verilog_header(conf *Config, arch *Arch, flavor string) string {
	result := ""
	return result
}

func (op Jz) Op_instruction_verilog_state_machine(arch *Arch, flavor string) string {
	result := ""
	rom_word := arch.Max_word()
	opbits := arch.Opcodes_bits()

	reg_num := 1 << arch.R

	result += "					JZ: begin\n"
	if arch.R == 1 {
		result += "						case (rom_value[" + strconv.Itoa(rom_word-opbits-1) + "])\n"
	} else {
		result += "						case (rom_value[" + strconv.Itoa(rom_word-opbits-1) + ":" + strconv.Itoa(rom_word-opbits-int(arch.R)) + "])\n"
	}
	for i := 0; i < reg_num; i++ {
		result += "							" + strings.ToUpper(Get_register_name(i)) + " : begin\n"
		result += "								if(_" + strings.ToLower(Get_register_name(i)) + " == 'b0)\n"
		result += "									_pc <= #1 rom_value[" + strconv.Itoa(rom_word-opbits-1-int(arch.R)) + ":" + strconv.Itoa(rom_word-opbits-int(arch.O)-int(arch.R)) + "];\n"
		result += "								else\n"
		result += "									_pc <= #1 _pc + 1'b1;\n"
		result += "								$display(\"JZ " + strings.ToUpper(Get_register_name(i)) + " \",_" + strings.ToLower(Get_register_name(i)) + ");\n"
		result += "							end\n"
	}
	result += "						endcase\n"
	result += "					end\n"

	return result
}

func (op Jz) Op_instruction_verilog_footer(arch *Arch, flavor string) string {
	// TODO
	return ""
}

func (op Jz) Assembler(arch *Arch, words []string) (string, error) {
	opbits := arch.Opcodes_bits()
	rom_word := arch.Max_word()
	osize := int(arch.O)

	reg_num := 1 << arch.R

	if len(words) != 2 {
		return "", Prerror{"Wrong arguments number"}
	}

	result := ""
	for i := 0; i < reg_num; i++ {
		if words[0] == strings.ToLower(Get_register_name(i)) {
			result += zeros_prefix(int(arch.R), get_binary(i))
			break
		}
	}

	if result == "" {
		return "", Prerror{"Unknown register name " + words[0]}
	}

	if partial, err := Process_number(words[1]); err == nil {
		result += zeros_prefix(osize, partial)
	} else {
		return "", Prerror{err.Error()}
	}

	for i := opbits + int(arch.R) + osize; i < rom_word; i++ {
		result += "0"
	}

	return result, nil

}

func (op Jz) Disassembler(arch *Arch, instr string) (string, error) {
	osize := int(arch.O)
	reg_id := get_id(instr[:arch.R])
	result := strings.ToLower(Get_register_name(reg_id)) + " "
	value := get_id(instr[arch.R : int(arch.R)+osize])
	result += strconv.Itoa(value)
	return result, nil
}

// The simulation does nothing
func (op Jz) Simulate(vm *VM, instr string) error {
	// TODO
	vm.Pc = vm.Pc + 1
	return nil
}

// The random genaration does nothing
func (op Jz) Generate(arch *Arch) string {
	// TODO
	return ""
}

func (op Jz) Required_shared() (bool, []string) {
	// TODO
	return false, []string{}
}

func (op Jz) Required_modes() (bool, []string) {
	return false, []string{}
}

func (op Jz) Forbidden_modes() (bool, []string) {
	return false, []string{}
}

func (Op Jz) Op_instruction_verilog_reset(arch *Arch, flavor string) string {
	result := ""
	return result
}

func (Op Jz) Op_instruction_verilog_default_state(arch *Arch, flavor string) string {
	result := ""
	return result
}

func (Op Jz) Op_instruction_verilog_internal_state(arch *Arch, flavor string) string {
	return ""
}

func (Op Jz) Op_instruction_verilog_extra_modules(arch *Arch, flavor string) ([]string, []string) {
	return []string{}, []string{}
}

func (Op Jz) Abstract_Assembler(arch *Arch, words []string) ([]UsageNotify, error) {
	seq, types := Sequence_to_0(words[0])
	if len(seq) > 0 && types == O_REGISTER {

		result := make([]UsageNotify, 2)
		newnot0 := UsageNotify{C_OPCODE, "jz", I_NIL}
		result[0] = newnot0
		newnot1 := UsageNotify{C_REGSIZE, S_NIL, len(seq)}
		result[1] = newnot1

		return result, nil
	}

	return []UsageNotify{}, errors.New("Wrong register")
}

func (Op Jz) Op_instruction_verilog_extra_block(arch *Arch, flavor string, level uint8, blockname string, objects []string) string {
	result := ""
	switch blockname {
	default:
		result = ""
	}
	return result
}
