package mmapforge

var Magic = [4]byte{'M', 'M', 'F', 'G'}

const MagicString = "MMFG"

const Version uint32 = 1

const HeaderSize = 64

const StoreReserveVA = 1 << 30
