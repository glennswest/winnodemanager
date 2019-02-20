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
		reset_needed := false;
		result := check_ipv6();
		if (result == true){
			 reset_needed = true;
		   }
		if (reset_needed == true){
 			reset_needed = false;
 			PsReset();
 			}
}

func PsReset(){
	fmt.Printf("Reseting\n");
}

func check_ipv6() bool {
	thenamespace := "HKLM:\\SYSTEM\\CurrentControlSet\\Services\\Tcpip6\\Parameters";
	thevalue := GetPsRegValue(thenamespace,"DisabledComponents");
	if (thevalue == ""){ // Its not disabled}
	    // Lets Disable it
			fmt.Printf("Disable IPV6\n");
			SetPSRegValue(thenamespace,"DisabledComponents","0xffffffff","DWord");
      return(true);
		}
	return(false);
}

func GetLineArray(theresult string ,thelinenum int ) [] string {
	thelines := lines(theresult);
	output := standardizeSpaces(thelines[thelinenum]);
	varvals := strings.Split(output," ");
	return(varvals);
}

func GetPsRegValue(thenamespace string,thevaluename string) string {
	cmd := "Get-ItemProperty -Path " + thenamespace + " -Name '" + thevaluename + "' -ErrorAction SilentlyContinue";
	result := powershell(cmd);
	if (len(result) == 0){
		return "";
	  }
	varvals := GetLineArray(result,2);
	return(varvals[2]);
}

func SetPSRegValue(thenamespace string, thevaluename string, thevalue string, thetype string){
	thecmd := "New-ItemProperty -Path " + thenamespace + " -Name '" + thevaluename + "' -Value '" + thevalue + "' -PropertyType '" + thetype + "'";
  result := powershell(thecmd);
	fmt.Printf("SetPSRegValue: %s\n", result);
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
	theargs := strings.Split(thecmd," ");
	c,err := exec.Command("powershell", theargs...).CombinedOutput();
	cmd := string(c);

	if  err != nil {
	    return("");
    } else {
	    return(cmd);
    }
}
