package main;

import(
	    "fmt"
	    "os/exec"
			"strings"
	)

func main(){
  thever := GetWinVersion();
	fmt.Printf("Windows Version = %s\n",thever);
  check_windows_prereq();
}

func check_windows_prereq(){
		check_ipv6();
}

func check_ipv6(){
	regNamespace := "HKLM:\\SYSTEM\\CurrentControlSet\\Services\\Tcpip6\\Parameters";
  cmd := "Get-ItemProperty -Path " + regNamespace + " -Name 'DisabledComponents' -ErrorAction SilentlyContinue";
	result := powershell(cmd);
	fmt.Printf("ipv6: %s->\n%s\n",cmd,result)
}

func GetWinVersion() string {
// Major  Minor  Build  Revision
// -----  -----  -----  --------
// 10     0      17134  0
	result := powershell("[System.Environment]::OSVersion.Version");
  thelines := lines(result);

	verstr := standardizeSpaces(thelines[3]);
	va := strings.Split(verstr," ");
	ver := va[0] + "." + va[1] + "." + va[2] + "." + va[3];
	return(ver);
}


func lines(theval string) [] string {
	 values := strings.Split(strings.Replace(theval, "\r\n", "\n", -1), "\n");
	 return(values);
}

func standardizeSpaces(s string) string {
    return strings.Join(strings.Fields(s), " ")
}

func powershell(thecmd string) string {
	c,err := exec.Command("powershell", thecmd).CombinedOutput();
	cmd := string(c);

	if  err != nil {
	    return("");
    } else {
	    return(cmd);
    }
}
