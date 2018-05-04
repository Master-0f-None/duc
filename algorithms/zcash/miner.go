package zcash

import (
	"fmt"
	"log"
	"time"
	"unsafe"
	"github.com/kilo17/go-opencl/cl"
	"github.com/kilo17/GoEndian"
//	"os"

 	"encoding/hex"
	"github.com/kilo17/awesomeProject/mining"

	"os"

	"crypto/sha256"
	"hash"

)


type CSha256 struct {
	state		[8]uint32
	count		uint64
	buffer		[64]uint8
}


// Miner actually mines :-)
type Miner struct {
	ClDevices       map[int]*cl.Device
	HashRateReports chan *mining.HashRateReport
//	Client          clients.Client
}

//singleDeviceMiner actually mines on 1 opencl device
type singleDeviceMiner struct {
	ClDevice        *cl.Device
	MinerID         int
	HashRateReports chan *mining.HashRateReport
//	Client          clients.Client
}

//Mine spawns a separate miner for each device defined in the CLDevices and feeds it with work
func (m *Miner) Mine() {
	log.Println("66666 ")
//	m.Client.Start()
	for minerID, device := range m.ClDevices {
		sdm := &singleDeviceMiner{
			ClDevice:        device,
			MinerID:         minerID,
			HashRateReports: m.HashRateReports,
//			Client:          m.Client,
		}
		go sdm.mine()
	}
}

type Solst struct {
	nr             uint32
	likelyInvalids uint32
	valid          [maxSolutions]uint8
	Values         [maxSolutions][1 << equihashParamK]uint32
	Finalz		[512]uint32
}
//todo: changed


func numberOfComputeUnits(gpu string) int {
	//if gpu == "rx480" {
	//}
	if gpu == "Fiji" {
		return 64
	}

	log.Panicln("Unknown GPU: ", gpu)
	return 0
}

func selectWorkSizeBlake() (workSize int) {
	workSize =
		64 * /* thread per wavefront */
			blakeWPS * /* wavefront per simd */
			4 * /* simd per compute unit */
			numberOfComputeUnits("Fiji")
	// Make the work group size a multiple of the nr of wavefronts, while
	// dividing the number of inputs. This results in the worksize being a
	// power of 2.
	for (numberOfInputs % workSize) != 0 {
		workSize += 64
	}
	return
}


func (miner *singleDeviceMiner) mine() {


	rowsPerUint := 4
	if numberOfSlots < 16 {
		rowsPerUint = 8
	}

	log.Println(miner.MinerID, "- Initializing", miner.ClDevice.Type(), "-", miner.ClDevice.Name())

	context, err := cl.CreateContext([]*cl.Device{miner.ClDevice})
	if err != nil {
		log.Fatalln(miner.MinerID, "-", err)
	}
	defer context.Release()

	commandQueue, err := context.CreateCommandQueue(miner.ClDevice, 0)
	if err != nil {
		log.Fatalln(miner.MinerID, "-", err)
	}
	defer commandQueue.Release()

	program, err := context.CreateProgramWithSource([]string{kernelSource})
	if err != nil {
		log.Fatalln(miner.MinerID, "-", err)
	}
	defer program.Release()

	err = program.BuildProgram([]*cl.Device{miner.ClDevice}, "")
	if err != nil {
		log.Fatalln(miner.MinerID, "-", err)
	}

	//Create kernels
	kernelInitHt, err := program.CreateKernel("kernel_init_ht")
	if err != nil {
		log.Fatalln(miner.MinerID, "-", err)
	}
	defer kernelInitHt.Release()

	var kernelRounds [equihashParamK]*cl.Kernel
	for round := 0; round < equihashParamK; round++ {
		kernelRounds[round], err = program.CreateKernel(fmt.Sprintf("kernel_round%d", round))
		if err != nil {
			log.Fatalln(miner.MinerID, "-", err)
		}
		defer kernelRounds[round].Release()
	}

	kernelSolutions, err := program.CreateKernel("kernel_sols")
	if err != nil {
		log.Fatalln(miner.MinerID, "-", err)
	}
	defer kernelSolutions.Release()

	//Create memory buffers
	dbg := make([]byte, 8, 8)
	bufferDbg, err := context.CreateBufferUnsafe(cl.MemReadWrite|cl.MemCopyHostPtr, 8, unsafe.Pointer(&dbg[0]))
	if err != nil {
		log.Panicln(err)
	}
	defer bufferDbg.Release()

	var bufferHt [2]*cl.MemObject
	bufferHt[0] = mining.CreateEmptyBuffer(context, cl.MemReadWrite, htSize)
	defer bufferHt[0].Release()
	bufferHt[1] = mining.CreateEmptyBuffer(context, cl.MemReadWrite, htSize)
	defer bufferHt[1].Release()

	bufferSolutions := mining.CreateEmptyBuffer(context, cl.MemReadWrite, int(unsafe.Sizeof(Solst{})))
	defer bufferSolutions.Release()

	var bufferRowCounters [2]*cl.MemObject
	bufferRowCounters[0] = mining.CreateEmptyBuffer(context, cl.MemReadWrite, numberOfRows)
	defer bufferRowCounters[0].Release()
	bufferRowCounters[1] = mining.CreateEmptyBuffer(context, cl.MemReadWrite, numberOfRows)
	defer bufferRowCounters[1].Release()

	var xxx = "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
//	var xxx = "04000000e54c27544050668f272ec3b460e1cde745c6b21239a81dae637fde4704000000844bc0c55696ef9920eeda11c1eb41b0c2e7324b46cc2e7aa0c2aa7736448d7a000000000000000000000000000000000000000000000000000000000000000068241a587e7e061d250e000000000000010000000000000000000000000000000000000000000000"
	header := MustGet(xxx)
	tt := header[104:108]

	log.Println("bits ", tt)
	fmt.Printf("bits Hex %02x\n", tt)


/*
	fmt.Printf("Header 1#St9#1 - len=%d cap=%d slice=%v\n", len(header), cap(header), header) // X bytes approx. 140
	pp := header[0:4]
	qq := header[4:36]
	uu := header[36:68]
	rr := header[68:100]
	ss := header[100:104]
	tt := header[104:108]
	yy := header[108:128]
	zz := header[128:140]
	log.Println("version ", pp)
	log.Println("PrevHash ", qq)
	log.Println("Merkle ", uu)
	log.Println("reserved ", rr)
	log.Println("time ", ss)
	log.Println("bits ", tt)
	log.Println("nonce ", yy)
	log.Println("12 zero ", zz)
	fmt.Printf("version Hex %02x\n", pp)
	fmt.Printf("PrevHash Hex %02x\n", qq)
	fmt.Printf("Merkle Hex %02x\n", uu)
	fmt.Printf("reserved Hex %02x\n", rr)
	fmt.Printf("time Hex %02x\n", ss)
	fmt.Printf("bits Hex %02x\n", tt)
	fmt.Printf("nonce Hex %02x\n", yy)
	fmt.Printf("12 zero Hex %02x\n", zz)
	fmt.Printf("BigEndian Hex %02x\n", header)
*/

	for {
		start := time.Now()
//		target, header, deprecationChannel, job, err := miner.Client.GetHeaderForWork()
	//	log.Println("///////////////////////GetHeaderForWork//////////////////////////////////////////////////")
	//	log.Println("target", target )
	//	log.Println("deprecationChannel", deprecationChannel )
	//	log.Println("header", header )
	//	log.Println("job", job )
		if err != nil {
			log.Println("ERROR fetching work -", err)
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		continueMining := true
		if !continueMining {
			log.Println("Halting miner ", miner.MinerID)
			break
		}

	//	log.Println(target, header, deprecationChannel, job)
	//	log.Println("target", target)
	//	log.Println("Header 2#####2", header)
	//	log.Println("deprecationChannel",  deprecationChannel)
	//	log.Println("Header 3#job#3 ", job)				//todo REMEMBER Reserved "0"'s missing here

		// Process first BLAKE2b-400 block
		blake := &blake2b_state_t{}

	//	var blake2 [8]uint64
	//	for r := 0; r < 8; r++ {
	//		blake2[r] = blake.h[r]
	//		log.Println("blake2 -r", blake2[r])
	//	}





		zcash_blake2b_init(blake, zcashHashLength, equihashParamN, equihashParamK)
//		hashBlocksGeneric(blake, zcashHashLength, equihashParamN, equihashParamK)
//		for r := 0; r < 8; r++ {
//			log.Println("blake.h- init",blake)
//		}

//todo 	need to try bool 0
		zcash_blake2b_update_2(blake, false, header[:128])
//		for r := 0; r < 8; r++ {
//			log.Println("blake2 -r", blake)
//		}
//	zcash_blake2b_update(blake, header[:128], false)

//		zcash_blake2b_final(blake, out)


		bufferBlake, err := context.CreateBufferUnsafe(cl.MemReadOnly|cl.MemCopyHostPtr, 64, unsafe.Pointer(&blake.h[0]))
		if err != nil {
			log.Panicln(err)
		}
		var globalWorkgroupSize int
		var localWorkgroupSize int
		for round := 0; round < equihashParamK; round++ {
			// Now on every round!
			localWorkgroupSize = 256
			globalWorkgroupSize = numberOfRows / rowsPerUint
			kernelInitHt.SetArgBuffer(0, bufferHt[round%2])
			kernelInitHt.SetArgBuffer(1, bufferRowCounters[round%2])
			commandQueue.EnqueueNDRangeKernel(kernelInitHt, nil, []int{globalWorkgroupSize}, []int{localWorkgroupSize}, nil)

			if round == 0 {
				kernelRounds[round].SetArgBuffer(0, bufferBlake)
				kernelRounds[round].SetArgBuffer(1, bufferHt[round%2])
				kernelRounds[round].SetArgBuffer(2, bufferRowCounters[round%2])
				globalWorkgroupSize = selectWorkSizeBlake()
				kernelRounds[round].SetArgBuffer(3, bufferDbg)
			} else {
				kernelRounds[round].SetArgBuffer(0, bufferHt[(round-1)%2])
				kernelRounds[round].SetArgBuffer(1, bufferHt[round%2])
				kernelRounds[round].SetArgBuffer(2, bufferRowCounters[(round-1)%2])
				kernelRounds[round].SetArgBuffer(3, bufferRowCounters[round%2])
				globalWorkgroupSize = numberOfRows
				kernelRounds[round].SetArgBuffer(4, bufferDbg)
			}
			if round == equihashParamK-1 {
				kernelRounds[round].SetArgBuffer(5, bufferSolutions)
			}
			localWorkgroupSize = 64
			commandQueue.EnqueueNDRangeKernel(kernelRounds[round], nil, []int{globalWorkgroupSize}, []int{localWorkgroupSize}, nil)
		}
		kernelSolutions.SetArgBuffer(0, bufferHt[0])
		kernelSolutions.SetArgBuffer(1, bufferHt[1])
		kernelSolutions.SetArgBuffer(2, bufferSolutions)
		kernelSolutions.SetArgBuffer(3, bufferRowCounters[0])
		kernelSolutions.SetArgBuffer(4, bufferRowCounters[1])
		globalWorkgroupSize = numberOfRows
		commandQueue.EnqueueNDRangeKernel(kernelSolutions, nil, []int{globalWorkgroupSize}, []int{localWorkgroupSize}, nil)
		// read solutions

		solutionsFound := miner.verifySolutions(commandQueue, bufferSolutions, header)

		bufferBlake.Release()
		log.Println("Solutions found:", solutionsFound)


		hashRate := float64(solutionsFound) / (time.Since(start).Seconds() * 1000000)
		miner.HashRateReports <- &mining.HashRateReport{MinerID: miner.MinerID, HashRate: hashRate}
	}

}


func (miner *singleDeviceMiner) verifySolutions(commandQueue *cl.CommandQueue, bufferSolutions *cl.MemObject, header []byte) (solutionsFound int) {

	Sols := &Solst{}

	// Most OpenCL implementations of clEnqueueReadBuffer in blocking mode are
	// good, except Nvidia implementing it as a wasteful busy work.
	commandQueue.EnqueueReadBuffer(bufferSolutions, true, 0, int(unsafe.Sizeof(*Sols)), unsafe.Pointer(Sols), nil)

	log.Println("Before VERIFY SOLUTION/ out from blake:", Sols)

	// let's check these solutions we just read...
	if Sols.nr > maxSolutions {
		log.Printf("ERROR: %d (probably invalid) solutions were dropped!\n", Sols.nr-maxSolutions)
		Sols.nr = maxSolutions
	}
	for i := 0; uint32(i) < Sols.nr; i++ {

		solutionsFound := miner.verifySolution(Sols, i)
		log.Println("solutionsFound", solutionsFound)
	//	log.Println("Sols", Sols)
	//	log.Println("iiiii", i)
//		log.Println("Values Before Crunch:  ", Sols.Values)
//		log.Println("Values Before Crunch:  ", i)
	}


	miner.SubmitSolution(Sols, solutionsFound, header)

	return

}



func (miner *singleDeviceMiner) verifySolution(sols *Solst, index int) int{
	inputs := sols.Values[index]

//	var iiee []uint32
//	log.Println("0000000000000 verify sol DONE 0000000000000000000-")

	seenLength := (1 << (prefix + 1)) / 8
	seen := make([]uint8, seenLength, seenLength)
	var i uint32
	var tmp uint8
	// look for duplicate inputs
	for i = 0; i < (1 << equihashParamK); i++ {
		if inputs[i]/uint32(8) >= uint32(seenLength) {
			log.Printf("Invalid input retrieved from device: %d\n", inputs[i])
			sols.valid[index] = 0
			return 0
		}
		tmp = seen[inputs[i]/8]
		seen[inputs[i]/8] |= 1 << (inputs[i] & 7)
		if tmp == seen[inputs[i]/8] {
			// at least one input value is a duplicate
			sols.valid[index] = 0
			return 0
		}
	}
	// the valid flag is already set by the GPU, but set it again because
	// I plan to change the GPU code to not set it
	sols.valid[index] = 1
	// sort the pairs in place
//	log.Println("111111111111 -sort pair DONE 111111111111111")

	for level := 0; level < equihashParamK; level++ {
		for i := 0; i < (1 << equihashParamK); i += 2 << uint(level) {

			len := 1 << uint(level)


		sortPair(inputs[i:i+len], inputs[i+len:i+(2*len)])
		//	sols.Finalz[i] = inputs[i]


		}
	}

//	log.Println("bbbbbbbbbbbbbbbbb", inputs)
	for j := range inputs {
	sols.Finalz[j] = inputs[j]

	}


///	log.Println("pppppppppppppppppppppppppppppppppppppp", sols.Finalz)
	return 1

}

func sortPair(a, b []uint32){
	needSorting := false
//	log.Println("SORT PAIRS: ", a)//todo////////////////////////////////////////////////////////////////////////////////
//	log.Println("111111111111111111111111111")

	var tmp uint32
	for i := 0; i < len(a); i++ {
		if needSorting || a[i] > b[i] {
			needSorting = true
			tmp = a[i]
			a[i] = b[i]
			b[i] = tmp
		} else {
			if a[i] < b[i] {
				return
			}
		}

	}




}
//todo: changed
//todo 				miner.SubmitSolutionZEC(sols, solutionsFound, header, target, job)

//todo  out		ZCASH_SOL_LEN-byte buffer where the solution will be stored= 1344 uint8_t
//todo  inputs		array of 32-bit inputs
//todo   n		number of elements in array = 512

func (miner *singleDeviceMiner) SubmitSolution(Solutions *Solst, solutionsFound int, header []byte) {
	var Finally string
	sliceExtract := make([]byte, 0)

	for i := 0; i < int(Solutions.nr); i++ {
		if Solutions.valid[i] > 0 {
			/*
			log.Println("22222222222222222222")
			log.Println("44444444444  calls store_encoded_sol DONE 444444444444444444")
			log.Println("5555555 Store Encoded Solution DONE 555555555555555")
			log.Println("Solutions Values ", Solutions.Values[i])
			log.Println("Solutions.Finalz)  ", Solutions.Finalz)
			log.Println("Values ", Solutions.Values[i])
			log.Println("Finalz  ", Solutions.Finalz[i])
			log.Println("Values Length ", len(Solutions.Values[i]))
			log.Println("Finalz Length ", len(Solutions.Finalz))
*/
			var inputs= Solutions.Finalz

			//		var inputs= [16]uint32{3257, 933560, 120482, 855408, 926622, 2063223, 1493414, 2060455, 128849, 1486455, 216187, 1287572, 308040, 1320625, 2022698, 2085613}

			//		var n uint32 = 8

			var n uint32 = 512

			var byte_pos uint32 = 0
			var bits_left uint32 = prefix + 1
			var x uint32 = 0
			var x_bits_used uint32 = 0
			slice := make([]uint32, 0)
			const MaxUint= ^uint32(0)

			for ; byte_pos < n; {

				if bits_left >= 8-x_bits_used {

					x |= inputs[byte_pos] >> (bits_left - 8 + x_bits_used)
					bits_left -= 8 - x_bits_used
					x_bits_used = 8

				} else if bits_left > 0 {
					var mask uint32 = ^(MaxUint << (8 - x_bits_used))
					mask = ((^mask) >> bits_left) & mask
					x |= (inputs[byte_pos] << (8 - x_bits_used - bits_left)) & mask
					x_bits_used += bits_left
					bits_left = 0
				} else if bits_left <= 0 {
					byte_pos++
					bits_left = prefix + 1

				}
				if x_bits_used == 8 {
					slice = append(slice, x)
					x = 0
					x_bits_used = 0
				}
			}

			//		log.Println("66666666 Store Encoded Solution DONE 666666666666666")

			for v := range slice {

				var Extract uint32 = slice[v]                        //todo
				Extract4th := make([]byte, 4)                        //todo
				endian.Endian.PutUint32(Extract4th, uint32(Extract)) //todo
				sliceExtract = append(sliceExtract, Extract4th[0])

				//	log.Println("BOTTOM OF MINER.GO", Finally)

			}


			Finally = fmt.Sprintf("%02x\n", sliceExtract)

			log.Println("BOTTOM OF MINER.GO", Finally)



	//	sliceExtract = sliceExtract[:len(sliceExtract)-1]


			stringz := string("0000000083fd0000000000000000000000000000000000000000000000000000")
		//	stringz := string("0020c49ba5e353f7ced916872b020c49ba5e353f7ced916872b020c49ba5e353")
			target := MustGet(stringz)



			log.Println("nnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn", header)
			log.Println("oooooooooooooooooooooooooooooo", len(header))
			fmt.Printf("pppppppppppppppppppppppppp    %x\n", header)

			slice1 := make([]uint8, len(header))
			copy(slice1, header)

			var slice2 = []uint8("\xfd\x40\x05")
			var slice3= sliceExtract
			var slice4 []uint8
			slice4 = append(slice4, slice1...)
			slice4 = append(slice4, slice2...)
			slice4 = append(slice4, slice3...)

			fmt.Printf("slice4   %x\n", slice4)
			fmt.Printf("sliceExtract   %x\n", sliceExtract)

			Sha256 := DoubleSHA(slice4)


		//	h := (uint32)(((fh.year*100+fh.month)*100+fh.day)*100 + fh.h)
		//	a := make([]byte, 4)
		//	binary.LittleEndian.PutUint32(a, h)

			fmt.Printf("target   %x\n", target)
			fmt.Printf("SHa256   %x\n", Sha256)
		//	log.Println("SHa2564   ", Sha2564)

			Sha256Rev := reverse(Sha256)


			okiedokie := cmp_target_256(target, Sha256Rev)

fmt.Printf("header---------------------%x\n", header[113:140])



			if okiedokie < 0 {

				log.Println("Hash is above target")
				os.Exit(3)

				return
			}else {
				log.Println("Hash is under target")
				fmt.Printf("oooooooooooooo-sliceExtract-oooooooooooooooo   %x\n  ", sliceExtract)

				os.Exit(3)
			}
		}


		}

	}

/*

*/

func MustGet(value string) []byte {
	i, err := hex.DecodeString(value)

	if err != nil {
		panic(err)
	}
	return i
}

func DoubleSHA(b []byte)([]byte){
	var h hash.Hash = sha256.New()
	h.Write(b)
	var h2 hash.Hash = sha256.New()
	h2.Write(h.Sum(nil))

	return h2.Sum(nil)
}

func Doublesha256(data []byte) string {
	hash := hash256(hash256(data))
//	return hash
	return hex.EncodeToString(hash)
}

func hash256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}


func cmp_target_256(a []uint8, b []uint8) int32 {
	log.Println("target", a)
	log.Println("sha256", b)


	for i := 0; i < 32; i++ {
		if a[i] != b[i] {
			var ddd = int32(a[i]) - int32(b[i])



			log.Println("ddd", ddd)

			return ddd
		}
	}
	return 0
}

func reverse(numbers []uint8) []uint8{
	newNumbers := make([]uint8, len(numbers))
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		newNumbers[i], newNumbers[j] = numbers[j], numbers[i]
	}
	return newNumbers
}

/*
	//			go func() {
			//
			//				if e := miner.Client.SubmitSolution(Finally, solutionsFound, header, target, job); e != nil {
			//					log.Println(miner.MinerID, "- Error submitting solution -", e)
			//				}
			//			}()


var valA = int(a[i])
			var valB = int(b[i])

			textA := strconv.Itoa(valA)
			textB:= strconv.Itoa(valB)

			valuesA = append(valuesA, textA)
			valuesB = append(valuesB, textB)
		}

		valuesAfinal := strings.Join(valuesA, "")
	valuesBfinal := strings.Join(valuesB, "")
	number, _ := strconv.ParseInt(a, 16, 32)

log.Println("valuesAfinal", valuesAfinal)
	log.Println("valuesBfinal", valuesBfinal)
	log.Println("aa", number)


	return 5




// todo inputs = Solutions.Finalz
// todo	 out = 	Finally(hex) or sliceExtract bytes
// todo


for i := SHA256_TARGET_LEN - 1; i >= 0; i-- {
var u= uint32(target[i])
var s= strconv.FormatUint(uint64(u), 10)

gg = append(gg, s)

}

log.Println("ggggggggggggggggg", gg)

//verify_sols(cl_command_queue queue, cl_mem buf_sols, uint64_t *nonce,
////sort_pair(&inputs[i], 1 << level);
//print_sols(sols_t *all_sols, uint64_t *nonce, uint32_t nr_valid_sols, uint8_t *header, size_t fixed_nonce_bytes, uint8_t *target, char *job_id)
//print_solver_line(inputs, header, fixed_nonce_bytes, target, job_id);
//	//TODO	DONE ABOVE -store_encoded_sol(p, values, 1 << PARAM_K); --aka SubmitSolution
///		-cmp_target_256

*/



