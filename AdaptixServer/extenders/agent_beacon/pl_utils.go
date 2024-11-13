package main

import (
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"io"
	"net"
	"strings"
)

const (
	COMMAND_CD        = 8
	COMMAND_CP        = 12
	COMMAND_DOWNLOAD  = 32
	COMMAND_PWD       = 4
	COMMAND_TERMINATE = 10

	COMMAND_ERROR = 0x1111ffff
)

const (
	DOWNLOAD_START    = 0x1
	DOWNLOAD_CONTINUE = 0x2
	DOWNLOAD_FINISH   = 0x3

	DOWNLOAD_STATE_RUNNING  = 0x1
	DOWNLOAD_STATE_STOPPED  = 0x2
	DOWNLOAD_STATE_FINISHED = 0x3
	DOWNLOAD_STATE_CANCELED = 0x4
)

var codePageMapping = map[int]*charmap.Charmap{
	037:   charmap.CodePage037,  // IBM EBCDIC US-Canada
	437:   charmap.CodePage437,  // OEM United States
	850:   charmap.CodePage850,  // Western European (DOS)
	852:   charmap.CodePage852,  // Central European (DOS)
	855:   charmap.CodePage855,  // OEM Cyrillic (primarily Russian)
	858:   charmap.CodePage858,  // OEM Multilingual Latin 1 + Euro
	860:   charmap.CodePage860,  // Portuguese (DOS)
	862:   charmap.CodePage862,  // Hebrew (DOS)
	863:   charmap.CodePage863,  // French Canadian (DOS)
	865:   charmap.CodePage865,  // Nordic (DOS)
	866:   charmap.CodePage866,  // Russian (DOS)
	1047:  charmap.CodePage1047, // IBM EBCDIC Latin 1/Open System
	1140:  charmap.CodePage1140, // IBM EBCDIC US-Canada with Euro
	1250:  charmap.Windows1250,  // Central European (Windows)
	1251:  charmap.Windows1251,  // Cyrillic (Windows)
	1252:  charmap.Windows1252,  // Western European (Windows)
	1253:  charmap.Windows1253,  // Greek (Windows)
	1254:  charmap.Windows1254,  // Turkish (Windows)
	1255:  charmap.Windows1255,  // Hebrew (Windows)
	1256:  charmap.Windows1256,  // Arabic (Windows)
	1257:  charmap.Windows1257,  // Baltic (Windows)
	1258:  charmap.Windows1258,  // Vietnamese (Windows)
	20866: charmap.KOI8R,        // Russian (KOI8-R)
	21866: charmap.KOI8U,        // Ukrainian (KOI8-U)
	28591: charmap.ISO8859_1,    // Western European (ISO 8859-1)
	28592: charmap.ISO8859_2,    // Central European (ISO 8859-2)
	28593: charmap.ISO8859_3,    // Latin 3 (ISO 8859-3)
	28594: charmap.ISO8859_4,    // Baltic (ISO 8859-4)
	28595: charmap.ISO8859_5,    // Cyrillic (ISO 8859-5)
	28596: charmap.ISO8859_6,    // Arabic (ISO 8859-6)
	28597: charmap.ISO8859_7,    // Greek (ISO 8859-7)
	28598: charmap.ISO8859_8,    // Hebrew (ISO 8859-8)
	28599: charmap.ISO8859_9,    // Turkish (ISO 8859-9)
	28605: charmap.ISO8859_15,   // Latin 9 (ISO 8859-15)
}

var win32ErrorCodes = map[uint]string{
	1:    "INVALID_FUNCTION",
	2:    "FILE_NOT_FOUND",
	3:    "PATH_NOT_FOUND",
	4:    "TOO_MANY_OPEN_FILES",
	5:    "ACCESS_DENIED",
	6:    "INVALID_HANDLE",
	7:    "ARENA_TRASHED",
	8:    "NOT_ENOUGH_MEMORY",
	9:    "INVALID_BLOCK",
	10:   "BAD_ENVIRONMENT",
	11:   "BAD_FORMAT",
	12:   "INVALID_ACCESS",
	13:   "INVALID_DATA",
	14:   "OUTOFMEMORY",
	15:   "INVALID_DRIVE",
	16:   "CURRENT_DIRECTORY",
	17:   "NOT_SAME_DEVICE",
	18:   "NO_MORE_FILES",
	19:   "WRITE_PROTECT",
	20:   "BAD_UNIT",
	21:   "NOT_READY",
	22:   "BAD_COMMAND",
	23:   "CRC",
	24:   "BAD_LENGTH",
	25:   "SEEK",
	26:   "NOT_DOS_DISK",
	27:   "SECTOR_NOT_FOUND",
	28:   "OUT_OF_PAPER",
	29:   "WRITE_FAULT",
	30:   "READ_FAULT",
	31:   "GEN_FAILURE",
	32:   "SHARING_VIOLATION",
	33:   "LOCK_VIOLATION",
	34:   "WRONG_DISK",
	36:   "SHARING_BUFFER_EXCEEDED",
	38:   "HANDLE_EOF",
	39:   "HANDLE_DISK_FULL",
	50:   "NOT_SUPPORTED",
	51:   "REM_NOT_LIST",
	52:   "DUP_NAME",
	53:   "BAD_NETPATH",
	54:   "NETWORK_BUSY",
	55:   "DEV_NOT_EXIST",
	56:   "TOO_MANY_CMDS",
	57:   "ADAP_HDW_ERR",
	58:   "BAD_NET_RESP",
	59:   "UNEXP_NET_ERR",
	60:   "BAD_REM_ADAP",
	61:   "PRINTQ_FULL",
	62:   "NO_SPOOL_SPACE",
	63:   "PRINT_CANCELLED",
	64:   "NETNAME_DELETED",
	65:   "NETWORK_ACCESS_DENIED",
	66:   "BAD_DEV_TYPE",
	67:   "BAD_NET_NAME",
	68:   "TOO_MANY_NAMES",
	69:   "TOO_MANY_SESS",
	70:   "SHARING_PAUSED",
	71:   "REQ_NOT_ACCEP",
	72:   "REDIR_PAUSED",
	80:   "FILE_EXISTS",
	82:   "CANNOT_MAKE",
	83:   "FAIL_I24",
	84:   "OUT_OF_STRUCTURES",
	85:   "ALREADY_ASSIGNED",
	86:   "INVALID_PASSWORD",
	87:   "INVALID_PARAMETER",
	88:   "NET_WRITE_FAULT",
	89:   "NO_PROC_SLOTS",
	100:  "TOO_MANY_SEMAPHORES",
	101:  "EXCL_SEM_ALREADY_OWNED",
	102:  "SEM_IS_SET",
	103:  "TOO_MANY_SEM_REQUESTS",
	104:  "INVALID_AT_INTERRUPT_TIME",
	105:  "SEM_OWNER_DIED",
	106:  "SEM_USER_LIMIT",
	107:  "DISK_CHANGE",
	108:  "DRIVE_LOCKED",
	109:  "BROKEN_PIPE",
	110:  "OPEN_FAILED",
	111:  "BUFFER_OVERFLOW",
	112:  "DISK_FULL",
	113:  "NO_MORE_SEARCH_HANDLES",
	114:  "INVALID_TARGET_HANDLE",
	117:  "INVALID_CATEGORY",
	118:  "INVALID_VERIFY_SWITCH",
	119:  "BAD_DRIVER_LEVEL",
	120:  "CALL_NOT_IMPLEMENTED",
	121:  "SEM_TIMEOUT",
	122:  "INSUFFICIENT_BUFFER",
	123:  "INVALID_NAME",
	124:  "INVALID_LEVEL",
	125:  "NO_VOLUME_LABEL",
	126:  "MOD_NOT_FOUND",
	127:  "PROC_NOT_FOUND",
	128:  "WAIT_NO_CHILDREN",
	129:  "CHILD_NOT_COMPLETE",
	130:  "DIRECT_ACCESS_HANDLE",
	131:  "NEGATIVE_SEEK",
	132:  "SEEK_ON_DEVICE",
	133:  "IS_JOIN_TARGET",
	134:  "IS_JOINED",
	135:  "IS_SUBSTED",
	136:  "NOT_JOINED",
	137:  "NOT_SUBSTED",
	138:  "JOIN_TO_JOIN",
	139:  "SUBST_TO_SUBST",
	140:  "JOIN_TO_SUBST",
	141:  "SUBST_TO_JOIN",
	142:  "BUSY_DRIVE",
	143:  "SAME_DRIVE",
	144:  "DIR_NOT_ROOT",
	145:  "DIR_NOT_EMPTY",
	146:  "IS_SUBST_PATH",
	147:  "IS_JOIN_PATH",
	148:  "PATH_BUSY",
	149:  "IS_SUBST_TARGET",
	150:  "SYSTEM_TRACE",
	151:  "INVALID_EVENT_COUNT",
	152:  "TOO_MANY_MUXWAITERS",
	153:  "INVALID_LIST_FORMAT",
	154:  "LABEL_TOO_LONG",
	155:  "TOO_MANY_TCBS",
	156:  "SIGNAL_REFUSED",
	157:  "DISCARDED",
	158:  "NOT_LOCKED",
	159:  "BAD_THREADID_ADDR",
	160:  "BAD_ARGUMENTS",
	161:  "BAD_PATHNAME",
	162:  "SIGNAL_PENDING",
	164:  "MAX_THRDS_REACHED",
	167:  "LOCK_FAILED",
	170:  "BUSY",
	173:  "CANCEL_VIOLATION",
	174:  "ATOMIC_LOCKS_NOT_SUPPORTED",
	180:  "INVALID_SEGMENT_NUMBER",
	182:  "INVALID_ORDINAL",
	183:  "ALREADY_EXISTS",
	186:  "INVALID_FLAG_NUMBER",
	187:  "SEM_NOT_FOUND",
	188:  "INVALID_STARTING_CODESEG",
	189:  "INVALID_STACKSEG",
	190:  "INVALID_MODULETYPE",
	191:  "INVALID_EXE_SIGNATURE",
	192:  "EXE_MARKED_INVALID",
	193:  "BAD_EXE_FORMAT",
	194:  "ITERATED_DATA_EXCEEDS_64k",
	195:  "INVALID_MINALLOCSIZE",
	196:  "DYNLINK_FROM_INVALID_RING",
	197:  "IOPL_NOT_ENABLED",
	198:  "INVALID_SEGDPL",
	199:  "AUTODATASEG_EXCEEDS_64k",
	200:  "RING2SEG_MUST_BE_MOVABLE",
	201:  "RELOC_CHAIN_XEEDS_SEGLIM",
	202:  "INFLOOP_IN_RELOC_CHAIN",
	203:  "ENVVAR_NOT_FOUND",
	205:  "NO_SIGNAL_SENT",
	206:  "FILENAME_EXCED_RANGE",
	207:  "RING2_STACK_IN_USE",
	208:  "META_EXPANSION_TOO_LONG",
	209:  "INVALID_SIGNAL_NUMBER",
	210:  "THREAD_1_INACTIVE",
	212:  "LOCKED",
	214:  "TOO_MANY_MODULES",
	215:  "NESTING_NOT_ALLOWED",
	216:  "EXE_MACHINE_TYPE_MISMATCH",
	217:  "EXE_CANNOT_MODIFY_SIGNED_BINARY",
	218:  "EXE_CANNOT_MODIFY_STRONG_SIGNED_BINARY",
	220:  "FILE_CHECKED_OUT",
	221:  "CHECKOUT_REQUIRED",
	222:  "BAD_FILE_TYPE",
	223:  "FILE_TOO_LARGE",
	224:  "FORMS_AUTH_REQUIRED",
	225:  "VIRUS_INFECTED",
	226:  "VIRUS_DELETED",
	229:  "PIPE_LOCAL",
	230:  "BAD_PIPE",
	231:  "PIPE_BUSY",
	232:  "NO_DATA",
	233:  "PIPE_NOT_CONNECTED",
	234:  "MORE_DATA",
	240:  "VC_DISCONNECTED",
	254:  "INVALID_EA_NAME",
	255:  "EA_LIST_INCONSISTENT",
	258:  "WAIT_TIMEOUT",
	259:  "NO_MORE_ITEMS",
	266:  "CANNOT_COPY",
	267:  "DIRECTORY",
	275:  "EAS_DIDNT_FIT",
	276:  "EA_FILE_CORRUPT",
	277:  "EA_TABLE_FULL",
	278:  "INVALID_EA_HANDLE",
	282:  "EAS_NOT_SUPPORTED",
	288:  "NOT_OWNER",
	298:  "TOO_MANY_POSTS",
	299:  "PARTIAL_COPY",
	300:  "OPLOCK_NOT_GRANTED",
	301:  "INVALID_OPLOCK_PROTOCOL",
	302:  "DISK_TOO_FRAGMENTED",
	303:  "DELETE_PENDING",
	317:  "MR_MID_NOT_FOUND",
	318:  "SCOPE_NOT_FOUND",
	350:  "FAIL_NOACTION_REBOOT",
	351:  "FAIL_SHUTDOWN",
	352:  "FAIL_RESTART",
	353:  "MAX_SESSIONS_REACHED",
	400:  "THREAD_MODE_ALREADY_BACKGROUND",
	401:  "THREAD_MODE_NOT_BACKGROUND",
	402:  "PROCESS_MODE_ALREADY_BACKGROUND",
	403:  "PROCESS_MODE_NOT_BACKGROUND",
	487:  "INVALID_ADDRESS",
	500:  "USER_PROFILE_LOAD",
	534:  "ARITHMETIC_OVERFLOW",
	535:  "PIPE_CONNECTED",
	536:  "PIPE_LISTENING",
	537:  "VERIFIER_STOP",
	538:  "ABIOS_ERROR",
	539:  "WX86_WARNING",
	540:  "WX86_ERROR",
	541:  "TIMER_NOT_CANCELED",
	542:  "UNWIND",
	543:  "BAD_STACK",
	544:  "INVALID_UNWIND_TARGET",
	545:  "INVALID_PORT_ATTRIBUTES",
	546:  "PORT_MESSAGE_TOO_LONG",
	547:  "INVALID_QUOTA_LOWER",
	548:  "DEVICE_ALREADY_ATTACHED",
	549:  "INSTRUCTION_MISALIGNMENT",
	550:  "PROFILING_NOT_STARTED",
	551:  "PROFILING_NOT_STOPPED",
	552:  "COULD_NOT_INTERPRET",
	553:  "PROFILING_AT_LIMIT",
	554:  "CANT_WAIT",
	555:  "CANT_TERMINATE_SELF",
	556:  "UNEXPECTED_MM_CREATE_ERR",
	557:  "UNEXPECTED_MM_MAP_ERROR",
	558:  "UNEXPECTED_MM_EXTEND_ERR",
	559:  "BAD_FUNCTION_TABLE",
	560:  "NO_GUID_TRANSLATION",
	561:  "INVALID_LDT_SIZE",
	563:  "INVALID_LDT_OFFSET",
	564:  "INVALID_LDT_DESCRIPTOR",
	565:  "TOO_MANY_THREADS",
	566:  "THREAD_NOT_IN_PROCESS",
	567:  "PAGEFILE_QUOTA_EXCEEDED",
	568:  "LOGON_SERVER_CONFLICT",
	569:  "SYNCHRONIZATION_REQUIRED",
	570:  "NET_OPEN_FAILED",
	571:  "IO_PRIVILEGE_FAILED",
	572:  "CONTROL_C_EXIT",
	573:  "MISSING_SYSTEMFILE",
	574:  "UNHANDLED_EXCEPTION",
	575:  "APP_INIT_FAILURE",
	576:  "PAGEFILE_CREATE_FAILED",
	577:  "INVALID_IMAGE_HASH",
	578:  "NO_PAGEFILE",
	579:  "ILLEGAL_FLOAT_CONTEXT",
	580:  "NO_EVENT_PAIR",
	581:  "DOMAIN_CTRLR_CONFIG_ERROR",
	582:  "ILLEGAL_CHARACTER",
	583:  "UNDEFINED_CHARACTER",
	584:  "FLOPPY_VOLUME",
	585:  "BIOS_FAILED_TO_CONNECT_INTERRUPT",
	586:  "BACKUP_CONTROLLER",
	587:  "MUTANT_LIMIT_EXCEEDED",
	588:  "FS_DRIVER_REQUIRED",
	589:  "CANNOT_LOAD_REGISTRY_FILE",
	590:  "DEBUG_ATTACH_FAILED",
	591:  "SYSTEM_PROCESS_TERMINATED",
	592:  "DATA_NOT_ACCEPTED",
	593:  "VDM_HARD_ERROR",
	594:  "DRIVER_CANCEL_TIMEOUT",
	595:  "REPLY_MESSAGE_MISMATCH",
	596:  "LOST_WRITEBEHIND_DATA",
	597:  "CLIENT_SERVER_PARAMETERS_INVALID",
	598:  "NOT_TINY_STREAM",
	599:  "STACK_OVERFLOW_READ",
	600:  "CONVERT_TO_LARGE",
	601:  "FOUND_OUT_OF_SCOPE",
	602:  "ALLOCATE_BUCKET",
	603:  "MARSHALL_OVERFLOW",
	604:  "INVALID_VARIANT",
	605:  "BAD_COMPRESSION_BUFFER",
	606:  "AUDIT_FAILED",
	607:  "TIMER_RESOLUTION_NOT_SET",
	608:  "INSUFFICIENT_LOGON_INFO",
	609:  "BAD_DLL_ENTRYPOINT",
	610:  "BAD_SERVICE_ENTRYPOINT",
	611:  "IP_ADDRESS_CONFLICT1",
	612:  "IP_ADDRESS_CONFLICT2",
	613:  "REGISTRY_QUOTA_LIMIT",
	614:  "NO_CALLBACK_ACTIVE",
	615:  "PWD_TOO_SHORT",
	616:  "PWD_TOO_RECENT",
	617:  "PWD_HISTORY_CONFLICT",
	618:  "UNSUPPORTED_COMPRESSION",
	619:  "INVALID_HW_PROFILE",
	620:  "INVALID_PLUGPLAY_DEVICE_PATH",
	621:  "QUOTA_LIST_INCONSISTENT",
	622:  "EVALUATION_EXPIRATION",
	623:  "ILLEGAL_DLL_RELOCATION",
	624:  "DLL_INIT_FAILED_LOGOFF",
	625:  "VALIDATE_CONTINUE",
	626:  "NO_MORE_MATCHES",
	627:  "RANGE_LIST_CONFLICT",
	628:  "SERVER_SID_MISMATCH",
	629:  "CANT_ENABLE_DENY_ONLY",
	630:  "FLOAT_MULTIPLE_FAULTS",
	631:  "FLOAT_MULTIPLE_TRAPS",
	632:  "NOINTERFACE",
	633:  "DRIVER_FAILED_SLEEP",
	634:  "CORRUPT_SYSTEM_FILE",
	635:  "COMMITMENT_MINIMUM",
	636:  "PNP_RESTART_ENUMERATION",
	637:  "SYSTEM_IMAGE_BAD_SIGNATURE",
	638:  "PNP_REBOOT_REQUIRED",
	639:  "INSUFFICIENT_POWER",
	640:  "MULTIPLE_FAULT_VIOLATION",
	641:  "SYSTEM_SHUTDOWN",
	642:  "PORT_NOT_SET",
	643:  "DS_VERSION_CHECK_FAILURE",
	644:  "RANGE_NOT_FOUND",
	646:  "NOT_SAFE_MODE_DRIVER",
	647:  "FAILED_DRIVER_ENTRY",
	648:  "DEVICE_ENUMERATION_ERROR",
	649:  "MOUNT_POINT_NOT_RESOLVED",
	650:  "INVALID_DEVICE_OBJECT_PARAMETER",
	651:  "MCA_OCCURED",
	652:  "DRIVER_DATABASE_ERROR",
	653:  "SYSTEM_HIVE_TOO_LARGE",
	654:  "DRIVER_FAILED_PRIOR_UNLOAD",
	655:  "VOLSNAP_PREPARE_HIBERNATE",
	656:  "HIBERNATION_FAILURE",
	665:  "FILE_SYSTEM_LIMITATION",
	668:  "ASSERTION_FAILURE",
	669:  "ACPI_ERROR",
	670:  "WOW_ASSERTION",
	671:  "PNP_BAD_MPS_TABLE",
	672:  "PNP_TRANSLATION_FAILED",
	673:  "PNP_IRQ_TRANSLATION_FAILED",
	674:  "PNP_INVALID_ID",
	675:  "WAKE_SYSTEM_DEBUGGER",
	676:  "HANDLES_CLOSED",
	677:  "EXTRANEOUS_INFORMATION",
	678:  "RXACT_COMMIT_NECESSARY",
	679:  "MEDIA_CHECK",
	680:  "GUID_SUBSTITUTION_MADE",
	681:  "STOPPED_ON_SYMLINK",
	682:  "LONGJUMP",
	683:  "PLUGPLAY_QUERY_VETOED",
	684:  "UNWIND_CONSOLIDATE",
	685:  "REGISTRY_HIVE_RECOVERED",
	686:  "DLL_MIGHT_BE_INSECURE",
	687:  "DLL_MIGHT_BE_INCOMPATIBLE",
	688:  "DBG_EXCEPTION_NOT_HANDLED",
	689:  "DBG_REPLY_LATER",
	690:  "DBG_UNABLE_TO_PROVIDE_HANDLE",
	691:  "DBG_TERMINATE_THREAD",
	692:  "DBG_TERMINATE_PROCESS",
	693:  "DBG_CONTROL_C",
	694:  "DBG_PRINTEXCEPTION_C",
	695:  "DBG_RIPEXCEPTION",
	696:  "DBG_CONTROL_BREAK",
	697:  "DBG_COMMAND_EXCEPTION",
	698:  "OBJECT_NAME_EXISTS",
	699:  "THREAD_WAS_SUSPENDED",
	700:  "IMAGE_NOT_AT_BASE",
	701:  "RXACT_STATE_CREATED",
	702:  "SEGMENT_NOTIFICATION",
	703:  "BAD_CURRENT_DIRECTORY",
	704:  "FT_READ_RECOVERY_FROM_BACKUP",
	705:  "FT_WRITE_RECOVERY",
	706:  "IMAGE_MACHINE_TYPE_MISMATCH",
	707:  "RECEIVE_PARTIAL",
	708:  "RECEIVE_EXPEDITED",
	709:  "RECEIVE_PARTIAL_EXPEDITED",
	710:  "EVENT_DONE",
	711:  "EVENT_PENDING",
	712:  "CHECKING_FILE_SYSTEM",
	713:  "FATAL_APP_EXIT",
	714:  "PREDEFINED_HANDLE",
	715:  "WAS_UNLOCKED",
	716:  "SERVICE_NOTIFICATION",
	717:  "WAS_LOCKED",
	718:  "LOG_HARD_ERROR",
	719:  "ALREADY_WIN32",
	720:  "IMAGE_MACHINE_TYPE_MISMATCH_EXE",
	721:  "NO_YIELD_PERFORMED",
	722:  "TIMER_RESUME_IGNORED",
	723:  "ARBITRATION_UNHANDLED",
	724:  "CARDBUS_NOT_SUPPORTED",
	725:  "MP_PROCESSOR_MISMATCH",
	726:  "HIBERNATED",
	727:  "RESUME_HIBERNATION",
	728:  "FIRMWARE_UPDATED",
	729:  "DRIVERS_LEAKING_LOCKED_PAGES",
	730:  "WAKE_SYSTEM",
	731:  "WAIT_1",
	732:  "WAIT_2",
	733:  "WAIT_3",
	734:  "WAIT_63",
	735:  "ABANDONED_WAIT_0",
	736:  "ABANDONED_WAIT_63",
	737:  "USER_APC",
	738:  "KERNEL_APC",
	739:  "ALERTED",
	740:  "ELEVATION_REQUIRED",
	741:  "REPARSE",
	742:  "OPLOCK_BREAK_IN_PROGRESS",
	743:  "VOLUME_MOUNTED",
	744:  "RXACT_COMMITTED",
	745:  "NOTIFY_CLEANUP",
	746:  "PRIMARY_TRANSPORT_CONNECT_FAILED",
	747:  "PAGE_FAULT_TRANSITION",
	748:  "PAGE_FAULT_DEMAND_ZERO",
	749:  "PAGE_FAULT_COPY_ON_WRITE",
	750:  "PAGE_FAULT_GUARD_PAGE",
	751:  "PAGE_FAULT_PAGING_FILE",
	752:  "CACHE_PAGE_LOCKED",
	753:  "CRASH_DUMP",
	754:  "BUFFER_ALL_ZEROS",
	755:  "REPARSE_OBJECT",
	756:  "RESOURCE_REQUIREMENTS_CHANGED",
	757:  "TRANSLATION_COMPLETE",
	758:  "NOTHING_TO_TERMINATE",
	759:  "PROCESS_NOT_IN_JOB",
	760:  "PROCESS_IN_JOB",
	761:  "VOLSNAP_HIBERNATE_READY",
	762:  "FSFILTER_OP_COMPLETED_SUCCESSFULLY",
	763:  "INTERRUPT_VECTOR_ALREADY_CONNECTED",
	764:  "INTERRUPT_STILL_CONNECTED",
	765:  "WAIT_FOR_OPLOCK",
	766:  "DBG_EXCEPTION_HANDLED",
	767:  "DBG_CONTINUE",
	768:  "CALLBACK_POP_STACK",
	769:  "COMPRESSION_DISABLED",
	770:  "CANTFETCHBACKWARDS",
	771:  "CANTSCROLLBACKWARDS",
	772:  "ROWSNOTRELEASED",
	773:  "BAD_ACCESSOR_FLAGS",
	774:  "ERRORS_ENCOUNTERED",
	775:  "NOT_CAPABLE",
	776:  "REQUEST_OUT_OF_SEQUENCE",
	777:  "VERSION_PARSE_ERROR",
	778:  "BADSTARTPOSITION",
	779:  "MEMORY_HARDWARE",
	780:  "DISK_REPAIR_DISABLED",
	781:  "INSUFFICIENT_RESOURCE_FOR_SPECIFIED_SHARED_SECTION_SIZE",
	782:  "SYSTEM_POWERSTATE_TRANSITION",
	783:  "SYSTEM_POWERSTATE_COMPLEX_TRANSITION",
	784:  "MCA_EXCEPTION",
	785:  "ACCESS_AUDIT_BY_POLICY",
	786:  "ACCESS_DISABLED_NO_SAFER_UI_BY_POLICY",
	787:  "ABANDON_HIBERFILE",
	788:  "LOST_WRITEBEHIND_DATA_NETWORK_DISCONNECTED",
	789:  "LOST_WRITEBEHIND_DATA_NETWORK_SERVER_ERROR",
	790:  "LOST_WRITEBEHIND_DATA_LOCAL_DISK_ERROR",
	791:  "BAD_MCFG_TABLE",
	994:  "EA_ACCESS_DENIED",
	995:  "OPERATION_ABORTED",
	996:  "IO_INCOMPLETE",
	997:  "IO_PENDING",
	998:  "NOACCESS",
	999:  "SWAPERROR",
	1001: "STACK_OVERFLOW",
	1002: "INVALID_MESSAGE",
	1003: "CAN_NOT_COMPLETE",
	1004: "INVALID_FLAGS",
	1005: "UNRECOGNIZED_VOLUME",
	1006: "FILE_INVALID",
	1007: "FULLSCREEN_MODE",
	1008: "NO_TOKEN",
	1009: "BADDB",
	1010: "BADKEY",
	1011: "CANTOPEN",
	1012: "CANTREAD",
	1013: "CANTWRITE",
	1014: "REGISTRY_RECOVERED",
	1015: "REGISTRY_CORRUPT",
	1016: "REGISTRY_IO_FAILED",
	1017: "NOT_REGISTRY_FILE",
	1018: "KEY_DELETED",
	1019: "NO_LOG_SPACE",
	1020: "KEY_HAS_CHILDREN",
	1021: "CHILD_MUST_BE_VOLATILE",
	1022: "NOTIFY_ENUM_DIR",
	1051: "DEPENDENT_SERVICES_RUNNING",
	1052: "INVALID_SERVICE_CONTROL",
	1053: "SERVICE_REQUEST_TIMEOUT",
	1054: "SERVICE_NO_THREAD",
	1055: "SERVICE_DATABASE_LOCKED",
	1056: "SERVICE_ALREADY_RUNNING",
	1057: "INVALID_SERVICE_ACCOUNT",
	1058: "SERVICE_DISABLED",
	1059: "CIRCULAR_DEPENDENCY",
	1060: "SERVICE_DOES_NOT_EXIST",
	1061: "SERVICE_CANNOT_ACCEPT_CTRL",
	1062: "SERVICE_NOT_ACTIVE",
	1063: "FAILED_SERVICE_CONTROLLER_CONNECT",
	1064: "EXCEPTION_IN_SERVICE",
	1065: "DATABASE_DOES_NOT_EXIST",
	1066: "SERVICE_SPECIFIC_ERROR",
	1067: "PROCESS_ABORTED",
	1068: "SERVICE_DEPENDENCY_FAIL",
	1069: "SERVICE_LOGON_FAILED",
	1070: "SERVICE_START_HANG",
	1071: "INVALID_SERVICE_LOCK",
	1072: "SERVICE_MARKED_FOR_DELETE",
	1073: "SERVICE_EXISTS",
	1074: "ALREADY_RUNNING_LKG",
	1075: "SERVICE_DEPENDENCY_DELETED",
	1076: "BOOT_ALREADY_ACCEPTED",
	1077: "SERVICE_NEVER_STARTED",
	1078: "DUPLICATE_SERVICE_NAME",
	1079: "DIFFERENT_SERVICE_ACCOUNT",
	1080: "CANNOT_DETECT_DRIVER_FAILURE",
	1081: "CANNOT_DETECT_PROCESS_ABORT",
	1082: "NO_RECOVERY_PROGRAM",
	1083: "SERVICE_NOT_IN_EXE",
	1084: "NOT_SAFEBOOT_SERVICE",
	1100: "END_OF_MEDIA",
	1101: "FILEMARK_DETECTED",
	1102: "BEGINNING_OF_MEDIA",
	1103: "SETMARK_DETECTED",
	1104: "NO_DATA_DETECTED",
	1105: "PARTITION_FAILURE",
	1106: "INVALID_BLOCK_LENGTH",
	1107: "DEVICE_NOT_PARTITIONED",
	1108: "UNABLE_TO_LOCK_MEDIA",
	1109: "UNABLE_TO_UNLOAD_MEDIA",
	1110: "MEDIA_CHANGED",
	1111: "BUS_RESET",
	1112: "NO_MEDIA_IN_DRIVE",
	1113: "NO_UNICODE_TRANSLATION",
	1114: "DLL_INIT_FAILED",
	1115: "SHUTDOWN_IN_PROGRESS",
	1116: "NO_SHUTDOWN_IN_PROGRESS",
	1117: "IO_DEVICE",
	1118: "SERIAL_NO_DEVICE",
	1119: "IRQ_BUSY",
	1120: "MORE_WRITES",
	1121: "COUNTER_TIMEOUT",
	1122: "FLOPPY_ID_MARK_NOT_FOUND",
	1123: "FLOPPY_WRONG_CYLINDER",
	1124: "FLOPPY_UNKNOWN_ERROR",
	1125: "FLOPPY_BAD_REGISTERS",
	1126: "DISK_RECALIBRATE_FAILED",
	1127: "DISK_OPERATION_FAILED",
	1128: "DISK_RESET_FAILED",
	1129: "EOM_OVERFLOW",
	1130: "NOT_ENOUGH_SERVER_MEMORY",
	1131: "POSSIBLE_DEADLOCK",
	1132: "MAPPED_ALIGNMENT",
	1140: "SET_POWER_STATE_VETOED",
	1141: "SET_POWER_STATE_FAILED",
	1142: "TOO_MANY_LINKS",
	1150: "OLD_WIN_VERSION",
	1151: "APP_WRONG_OS",
	1152: "SINGLE_INSTANCE_APP",
	1153: "RMODE_APP",
	1154: "INVALID_DLL",
	1155: "NO_ASSOCIATION",
	1156: "DDE_FAIL",
	1157: "DLL_NOT_FOUND",
	1158: "NO_MORE_USER_HANDLES",
	1159: "MESSAGE_SYNC_ONLY",
	1160: "SOURCE_ELEMENT_EMPTY",
	1161: "DESTINATION_ELEMENT_FULL",
	1162: "ILLEGAL_ELEMENT_ADDRESS",
	1163: "MAGAZINE_NOT_PRESENT",
	1164: "DEVICE_REINITIALIZATION_NEEDED",
	1165: "DEVICE_REQUIRES_CLEANING",
	1166: "DEVICE_DOOR_OPEN",
	1167: "DEVICE_NOT_CONNECTED",
	1168: "NOT_FOUND",
	1169: "NO_MATCH",
	1170: "SET_NOT_FOUND",
	1171: "POINT_NOT_FOUND",
	1172: "NO_TRACKING_SERVICE",
	1173: "NO_VOLUME_ID",
	2108: "CONNECTED_OTHER_PASSWORD",
	2202: "BAD_USERNAME",
	2250: "NOT_CONNECTED",
	2401: "OPEN_FILES",
	2402: "ACTIVE_CONNECTIONS",
	2404: "DEVICE_IN_USE",
	1200: "BAD_DEVICE",
	1201: "CONNECTION_UNAVAIL",
	1202: "DEVICE_ALREADY_REMEMBERED",
	1203: "NO_NET_OR_BAD_PATH",
	1204: "BAD_PROVIDER",
	1205: "CANNOT_OPEN_PROFILE",
	1206: "BAD_PROFILE",
	1207: "NOT_CONTAINER",
	1208: "EXTENDED_ERROR",
	1209: "INVALID_GROUPNAME",
	1210: "INVALID_COMPUTERNAME",
	1211: "INVALID_EVENTNAME",
	1212: "INVALID_DOMAINNAME",
	1213: "INVALID_SERVICENAME",
	1214: "INVALID_NETNAME",
	1215: "INVALID_SHARENAME",
	1216: "INVALID_PASSWORDNAME",
	1217: "INVALID_MESSAGENAME",
	1218: "INVALID_MESSAGEDEST",
	1219: "SESSION_CREDENTIAL_CONFLICT",
	1220: "REMOTE_SESSION_LIMIT_EXCEEDED",
	1221: "DUP_DOMAINNAME",
	1222: "NO_NETWORK",
	1223: "CANCELLED",
	1224: "USER_MAPPED_FILE",
	1225: "CONNECTION_REFUSED",
	1226: "GRACEFUL_DISCONNECT",
	1227: "ADDRESS_ALREADY_ASSOCIATED",
	1228: "ADDRESS_NOT_ASSOCIATED",
	1229: "CONNECTION_INVALID",
	1230: "CONNECTION_ACTIVE",
	1231: "NETWORK_UNREACHABLE",
	1232: "HOST_UNREACHABLE",
	1233: "PROTOCOL_UNREACHABLE",
	1234: "PORT_UNREACHABLE",
	1235: "REQUEST_ABORTED",
	1236: "CONNECTION_ABORTED",
	1237: "RETRY",
	1238: "CONNECTION_COUNT_LIMIT",
	1239: "LOGIN_TIME_RESTRICTION",
	1240: "LOGIN_WKSTA_RESTRICTION",
	1241: "INCORRECT_ADDRESS",
	1242: "ALREADY_REGISTERED",
	1243: "SERVICE_NOT_FOUND",
	1244: "NOT_AUTHENTICATED",
	1245: "NOT_LOGGED_ON",
	1246: "CONTINUE",
	1247: "ALREADY_INITIALIZED",
	1248: "NO_MORE_DEVICES",
	1249: "NO_SUCH_SITE",
	1250: "DOMAIN_CONTROLLER_EXISTS",
	1251: "DS_NOT_INSTALLED",
	1300: "NOT_ALL_ASSIGNED",
	1301: "SOME_NOT_MAPPED",
	1302: "NO_QUOTAS_FOR_ACCOUNT",
	1303: "LOCAL_USER_SESSION_KEY",
	1304: "NULL_LM_PASSWORD",
	1305: "UNKNOWN_REVISION",
	1306: "REVISION_MISMATCH",
	1307: "INVALID_OWNER",
	1308: "INVALID_PRIMARY_GROUP",
	1309: "NO_IMPERSONATION_TOKEN",
	1310: "CANT_DISABLE_MANDATORY",
	1311: "NO_LOGON_SERVERS",
	1312: "NO_SUCH_LOGON_SESSION",
	1313: "NO_SUCH_PRIVILEGE",
	1314: "PRIVILEGE_NOT_HELD",
	1315: "INVALID_ACCOUNT_NAME",
	1316: "USER_EXISTS",
	1317: "NO_SUCH_USER",
	1318: "GROUP_EXISTS",
	1319: "NO_SUCH_GROUP",
	1320: "MEMBER_IN_GROUP",
	1321: "MEMBER_NOT_IN_GROUP",
	1322: "LAST_ADMIN",
	1323: "WRONG_PASSWORD",
	1324: "ILL_FORMED_PASSWORD",
	1325: "PASSWORD_RESTRICTION",
	1326: "LOGON_FAILURE",
	1327: "ACCOUNT_RESTRICTION",
	1328: "INVALID_LOGON_HOURS",
	1329: "INVALID_WORKSTATION",
	1330: "PASSWORD_EXPIRED",
	1331: "ACCOUNT_DISABLED",
	1332: "NONE_MAPPED",
	1333: "TOO_MANY_LUIDS_REQUESTED",
	1334: "LUIDS_EXHAUSTED",
	1335: "INVALID_SUB_AUTHORITY",
	1336: "INVALID_ACL",
	1337: "INVALID_SID",
	1338: "INVALID_SECURITY_DESCR",
	1340: "BAD_INHERITANCE_ACL",
	1341: "SERVER_DISABLED",
	1342: "SERVER_NOT_DISABLED",
	1343: "INVALID_ID_AUTHORITY",
	1344: "ALLOTTED_SPACE_EXCEEDED",
	1345: "INVALID_GROUP_ATTRIBUTES",
	1346: "BAD_IMPERSONATION_LEVEL",
	1347: "CANT_OPEN_ANONYMOUS",
	1348: "BAD_VALIDATION_CLASS",
	1349: "BAD_TOKEN_TYPE",
	1350: "NO_SECURITY_ON_OBJECT",
	1351: "CANT_ACCESS_DOMAIN_INFO",
	1352: "INVALID_SERVER_STATE",
	1353: "INVALID_DOMAIN_STATE",
	1354: "INVALID_DOMAIN_ROLE",
	1355: "NO_SUCH_DOMAIN",
	1356: "DOMAIN_EXISTS",
	1357: "DOMAIN_LIMIT_EXCEEDED",
	1358: "INTERNAL_DB_CORRUPTION",
	1359: "INTERNAL_ERROR",
	1360: "GENERIC_NOT_MAPPED",
	1361: "BAD_DESCRIPTOR_FORMAT",
	1362: "NOT_LOGON_PROCESS",
	1363: "LOGON_SESSION_EXISTS",
	1364: "NO_SUCH_PACKAGE",
	1365: "BAD_LOGON_SESSION_STATE",
	1366: "LOGON_SESSION_COLLISION",
	1367: "INVALID_LOGON_TYPE",
	1368: "CANNOT_IMPERSONATE",
	1369: "RXACT_INVALID_STATE",
	1370: "RXACT_COMMIT_FAILURE",
	1371: "SPECIAL_ACCOUNT",
	1372: "SPECIAL_GROUP",
	1373: "SPECIAL_USER",
	1374: "MEMBERS_PRIMARY_GROUP",
	1375: "TOKEN_ALREADY_IN_USE",
	1376: "NO_SUCH_ALIAS",
	1377: "MEMBER_NOT_IN_ALIAS",
	1378: "MEMBER_IN_ALIAS",
	1379: "ALIAS_EXISTS",
	1380: "LOGON_NOT_GRANTED",
	1381: "TOO_MANY_SECRETS",
	1382: "SECRET_TOO_LONG",
	1383: "INTERNAL_DB_ERROR",
	1384: "TOO_MANY_CONTEXT_IDS",
	1385: "LOGON_TYPE_NOT_GRANTED",
	1386: "NT_CROSS_ENCRYPTION_REQUIRED",
	1387: "NO_SUCH_MEMBER",
	1388: "INVALID_MEMBER",
	1389: "TOO_MANY_SIDS",
	1390: "LM_CROSS_ENCRYPTION_REQUIRED",
	1391: "NO_INHERITANCE",
	1392: "FILE_CORRUPT",
	1393: "DISK_CORRUPT",
	1394: "NO_USER_SESSION_KEY",
	1395: "LICENSE_QUOTA_EXCEEDED",
	1400: "INVALID_WINDOW_HANDLE",
	1401: "INVALID_MENU_HANDLE",
	1402: "INVALID_CURSOR_HANDLE",
	1403: "INVALID_ACCEL_HANDLE",
	1404: "INVALID_HOOK_HANDLE",
	1405: "INVALID_DWP_HANDLE",
	1406: "TLW_WITH_WSCHILD",
	1407: "CANNOT_FIND_WND_CLASS",
	1408: "WINDOW_OF_OTHER_THREAD",
	1409: "HOTKEY_ALREADY_REGISTERED",
	1410: "CLASS_ALREADY_EXISTS",
	1411: "CLASS_DOES_NOT_EXIST",
	1412: "CLASS_HAS_WINDOWS",
	1413: "INVALID_INDEX",
	1414: "INVALID_ICON_HANDLE",
	1415: "PRIVATE_DIALOG_INDEX",
	1416: "LISTBOX_ID_NOT_FOUND",
	1417: "NO_WILDCARD_CHARACTERS",
	1418: "CLIPBOARD_NOT_OPEN",
	1419: "HOTKEY_NOT_REGISTERED",
	1420: "WINDOW_NOT_DIALOG",
	1421: "CONTROL_ID_NOT_FOUND",
	1422: "INVALID_COMBOBOX_MESSAGE",
	1423: "WINDOW_NOT_COMBOBOX",
	1424: "INVALID_EDIT_HEIGHT",
	1425: "DC_NOT_FOUND",
	1426: "INVALID_HOOK_FILTER",
	1427: "INVALID_FILTER_PROC",
	1428: "HOOK_NEEDS_HMOD",
	1429: "GLOBAL_ONLY_HOOK",
	1430: "JOURNAL_HOOK_SET",
	1431: "HOOK_NOT_INSTALLED",
	1432: "INVALID_LB_MESSAGE",
	1433: "SETCOUNT_ON_BAD_LB",
	1434: "LB_WITHOUT_TABSTOPS",
	1435: "DESTROY_OBJECT_OF_OTHER_THREAD",
	1436: "CHILD_WINDOW_MENU",
	1437: "NO_SYSTEM_MENU",
	1438: "INVALID_MSGBOX_STYLE",
	1439: "INVALID_SPI_VALUE",
	1440: "SCREEN_ALREADY_LOCKED",
	1441: "HWNDS_HAVE_DIFF_PARENT",
	1442: "NOT_CHILD_WINDOW",
	1443: "INVALID_GW_COMMAND",
	1444: "INVALID_THREAD_ID",
	1445: "NON_MDICHILD_WINDOW",
	1446: "POPUP_ALREADY_ACTIVE",
	1447: "NO_SCROLLBARS",
	1448: "INVALID_SCROLLBAR_RANGE",
	1449: "INVALID_SHOWWIN_COMMAND",
	1450: "NO_SYSTEM_RESOURCES",
	1451: "NONPAGED_SYSTEM_RESOURCES",
	1452: "PAGED_SYSTEM_RESOURCES",
	1453: "WORKING_SET_QUOTA",
	1454: "PAGEFILE_QUOTA",
	1455: "COMMITMENT_LIMIT",
	1456: "MENU_ITEM_NOT_FOUND",
	1457: "INVALID_KEYBOARD_HANDLE",
	1458: "HOOK_TYPE_NOT_ALLOWED",
	1459: "REQUIRES_INTERACTIVE_WINDOWSTATION",
	1460: "TIMEOUT",
	1461: "INVALID_MONITOR_HANDLE",
	1462: "INCORRECT_SIZE",
	1463: "SYMLINK_CLASS_DISABLED",
	1464: "SYMLINK_NOT_SUPPORTED",
	1465: "XML_PARSE_ERROR",
	1466: "XMLDSIG_ERROR",
	1467: "RESTART_APPLICATION",
	1468: "WRONG_COMPARTMENT",
	1469: "AUTHIP_FAILURE",
	1500: "EVENTLOG_FILE_CORRUPT",
	1501: "EVENTLOG_CANT_START",
	1502: "LOG_FILE_FULL",
	1503: "EVENTLOG_FILE_CHANGED",
	1601: "INSTALL_SERVICE",
	1602: "INSTALL_USEREXIT",
	1603: "INSTALL_FAILURE",
	1604: "INSTALL_SUSPEND",
	1605: "UNKNOWN_PRODUCT",
	1606: "UNKNOWN_FEATURE",
	1607: "UNKNOWN_COMPONENT",
	1608: "UNKNOWN_PROPERTY",
	1609: "INVALID_HANDLE_STATE",
	1610: "BAD_CONFIGURATION",
	1611: "INDEX_ABSENT",
	1612: "INSTALL_SOURCE_ABSENT",
	1613: "BAD_DATABASE_VERSION",
	1614: "PRODUCT_UNINSTALLED",
	1615: "BAD_QUERY_SYNTAX",
	1616: "INVALID_FIELD",
	1617: "DEVICE_REMOVED",
	1618: "INSTALL_ALREADY_RUNNING",
	1619: "INSTALL_PACKAGE_OPEN_FAILED",
	1620: "INSTALL_PACKAGE_INVALID",
	1621: "INSTALL_UI_FAILURE",
	1622: "INSTALL_LOG_FAILURE",
	1623: "INSTALL_LANGUAGE_UNSUPPORTED",
	1624: "INSTALL_TRANSFORM_FAILURE",
	1625: "INSTALL_PACKAGE_REJECTED",
	1626: "FUNCTION_NOT_CALLED",
	1627: "FUNCTION_FAILED",
	1628: "INVALID_TABLE",
	1629: "DATATYPE_MISMATCH",
	1630: "UNSUPPORTED_TYPE",
	1631: "CREATE_FAILED",
	1632: "INSTALL_TEMP_UNWRITABLE",
	1633: "INSTALL_PLATFORM_UNSUPPORTED",
	1634: "INSTALL_NOTUSED",
	1635: "PATCH_PACKAGE_OPEN_FAILED",
	1636: "PATCH_PACKAGE_INVALID",
	1637: "PATCH_PACKAGE_UNSUPPORTED",
	1638: "PRODUCT_VERSION",
	1639: "INVALID_COMMAND_LINE",
	1640: "INSTALL_REMOTE_DISALLOWED",
	1641: "SUCCESS_REBOOT_INITIATED",
	1642: "PATCH_TARGET_NOT_FOUND",
	1643: "PATCH_PACKAGE_REJECTED",
	1644: "INSTALL_TRANSFORM_REJECTED",
	1645: "INSTALL_REMOTE_PROHIBITED",
	1646: "PATCH_REMOVAL_UNSUPPORTED",
	1647: "UNKNOWN_PATCH",
	1648: "PATCH_NO_SEQUENCE",
	1649: "PATCH_REMOVAL_DISALLOWED",
	1650: "INVALID_PATCH_XML",
	1651: "PATCH_MANAGED_ADVERTISED_PRODUCT",
	1652: "INSTALL_SERVICE_SAFEBOOT",
}

func ConvertCpToUTF8(input string, codePage int) string {
	enc, exists := codePageMapping[codePage]
	if !exists {
		return input
	}

	reader := transform.NewReader(strings.NewReader(input), enc.NewDecoder())
	utf8Text, err := io.ReadAll(reader)
	if err != nil {
		return input
	}

	return string(utf8Text)
}

func ConvertUTF8toCp(input string, codePage int) string {
	enc, exists := codePageMapping[codePage]
	if !exists {
		return input
	}

	transform.NewWriter(io.Discard, enc.NewEncoder())
	encodedText, err := io.ReadAll(transform.NewReader(strings.NewReader(input), enc.NewEncoder()))
	if err != nil {
		return input
	}

	return string(encodedText)
}

func GetOsVersion(majorVersion uint8, minorVersion uint8, buildNumber uint, isServer bool, systemArch string) (int, string) {
	var (
		desc string
		os   = 0
	)

	osVersion := "unknown"
	if majorVersion == 10 && minorVersion == 0 && isServer && buildNumber >= 19045 {
		osVersion = "Win 2022 Serv"
	} else if majorVersion == 10 && minorVersion == 0 && isServer && buildNumber >= 17763 {
		osVersion = "Win 2019 Serv"
	} else if majorVersion == 10 && minorVersion == 0 && !isServer && buildNumber >= 22000 {
		osVersion = "Win 11"
	} else if majorVersion == 10 && minorVersion == 0 && isServer {
		osVersion = "Win 2016 Serv"
	} else if majorVersion == 10 && minorVersion == 0 {
		osVersion = "Win 10"
	} else if majorVersion == 6 && minorVersion == 3 && isServer {
		osVersion = "Win Serv 2012 R2"
	} else if majorVersion == 6 && minorVersion == 3 {
		osVersion = "Win 8.1"
	} else if majorVersion == 6 && minorVersion == 2 && isServer {
		osVersion = "Win Serv 2012"
	} else if majorVersion == 6 && minorVersion == 2 {
		osVersion = "Win 8"
	} else if majorVersion == 6 && minorVersion == 1 && isServer {
		osVersion = "Win Serv 2008 R2"
	} else if majorVersion == 6 && minorVersion == 1 {
		osVersion = "Win 7"
	}

	desc = osVersion + " " + systemArch
	if strings.HasSuffix(osVersion, "Win") {
		os = 1
	}
	return os, desc
}

func int32ToIPv4(ip uint) string {
	bytes := []byte{
		byte(ip),
		byte(ip >> 8),
		byte(ip >> 16),
		byte(ip >> 24),
	}
	return net.IP(bytes).String()
}
