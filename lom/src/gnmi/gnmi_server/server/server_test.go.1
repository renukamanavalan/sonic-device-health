package gnmi

// server_test covers gNMI get, subscribe (stream and poll) test
// Prerequisite: redis-server should be running.
import (
	"crypto/tls"
	"encoding/json"
	"path/filepath"
	"flag"
	"fmt"
"sync"
	"strings"
	"unsafe"

	testcert "github.com/sonic-net/sonic-gnmi/testdata/tls"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/proto"

	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"testing"
	"time"
	"runtime"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/client"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	ext_pb "github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/ygot"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/keepalive"

	// Register supported client types.
	spb "github.com/sonic-net/sonic-gnmi/proto"
	sgpb "github.com/sonic-net/sonic-gnmi/proto/gnoi"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	sdcfg "github.com/sonic-net/sonic-gnmi/sonic_db_config"
        "github.com/Workiva/go-datastructures/queue"
        linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/sonic-net/sonic-gnmi/common_utils"
	"github.com/sonic-net/sonic-gnmi/test_utils"
	gclient "github.com/jipanyang/gnmi/client/gnmi"
	"github.com/jipanyang/gnxi/utils/xpath"
	gnoi_system_pb "github.com/openconfig/gnoi/system"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/godbus/dbus/v5"
	cacheclient "github.com/openconfig/gnmi/client"

)

func TestXYX(t *testing.T) {
    t.Errorf("TODO: Need to rewrite tests")
}

func init() {
	// Enable logs at UT setup
	flag.Lookup("v").Value.Set("10")
	flag.Lookup("log_dir").Value.Set("/tmp/telemetrytest")
}
