/** 自定义 lint 规则统一契约:run 返回违例数,fix 为 true 时就地改写 */
export interface LintRule {
  name: string;
  run(fix: boolean): number;
}
